package inventory

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

type supplierFunc func(context.Context) ([]Product, error)

func (f supplierFunc) Fetch(ctx context.Context) ([]Product, error) { return f(ctx) }

func TestSyncIdempotentAndThresholdCrossing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	products := []Product{{SKU: "A", Name: "Coffee", Quantity: 15, PriceCents: 1000}, {SKU: "B", Name: "Filters", Quantity: 3, PriceCents: 300}}
	engine := &Engine{Source: supplierFunc(func(context.Context) ([]Product, error) { return products, nil }), Store: store, Threshold: 10}
	r, err := engine.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.Inserted != 2 || r.Alerts != 1 || r.Updated != 0 {
		t.Fatalf("initial run: %+v", r)
	}
	r, err = engine.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.Unchanged != 2 || r.Alerts != 0 || r.Updated != 0 {
		t.Fatalf("idempotent run: %+v", r)
	}
	products[0].Quantity = 4
	r, err = engine.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.Updated != 1 || r.Unchanged != 1 || r.Alerts != 1 {
		t.Fatalf("crossing run: %+v", r)
	}
	r, err = engine.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.Alerts != 0 || r.Unchanged != 2 {
		t.Fatalf("repeat run: %+v", r)
	}
	items, alerts, runs := store.Snapshot()
	if len(items) != 2 || len(alerts) != 2 || len(runs) != 4 {
		t.Fatalf("unexpected state %d %d %d", len(items), len(alerts), len(runs))
	}
	restored, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	restoredItems, restoredAlerts, restoredRuns := restored.Snapshot()
	if len(restoredItems) != 2 || len(restoredAlerts) != 2 || len(restoredRuns) != 4 {
		t.Fatal("persistence failed")
	}
}

func TestRejectInvalidSupplierPayloadWithoutPartialChanges(t *testing.T) {
	st, err := OpenStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{Store: st, Threshold: 10, Source: supplierFunc(func(context.Context) ([]Product, error) {
		return []Product{{SKU: "A", Name: "One", Quantity: 3}, {SKU: "A", Name: "Two", Quantity: 2}}, nil
	})}
	r, err := e.Sync(context.Background())
	if err == nil || r.Status != "failed" {
		t.Fatalf("want failed run: %+v, %v", r, err)
	}
	items, alerts, runs := st.Snapshot()
	if len(items) != 0 || len(alerts) != 0 || len(runs) != 1 || runs[0].Status != "failed" {
		t.Fatal("partial state was committed")
	}
	e.Source = supplierFunc(func(context.Context) ([]Product, error) { return nil, errors.New("temporary outage") })
	r, err = e.Sync(context.Background())
	if err == nil || r.Status != "failed" {
		t.Fatal("expected fetch error")
	}
}

func TestValidation(t *testing.T) {
	for _, p := range []Product{{SKU: "", Name: "x"}, {SKU: "X", Name: ""}, {SKU: "X", Name: "a", Quantity: -1}, {SKU: "X", Name: "a", PriceCents: -1}} {
		if p.Validate() == nil {
			t.Errorf("accepted invalid product %+v", p)
		}
	}
}
