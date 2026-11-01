package inventory

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestEndToEndAPI(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	mock := NewMockSupplier()
	engine := &Engine{Store: store, Threshold: 10}
	server := httptest.NewServer(NewHandler(engine, mock))
	defer server.Close()
	engine.Source = SupplierClient{URL: server.URL + "/mock/supplier/items"}
	res, err := http.Post(server.URL+"/api/sync", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("sync HTTP %d", res.StatusCode)
	}
	var run Run
	if err := json.NewDecoder(res.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if run.Inserted != 4 || run.Alerts != 1 {
		t.Fatalf("wrong initial sync: %+v", run)
	}
	patch, err := http.NewRequest(http.MethodPatch, server.URL+"/mock/supplier/items/COFFEE-001", bytes.NewBufferString(`{"quantity":4}`))
	if err != nil {
		t.Fatal(err)
	}
	patch.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(patch)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("patch HTTP %d", res.StatusCode)
	}
	res, err = http.Post(server.URL+"/api/sync", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Updated != 1 || run.Alerts != 1 {
		t.Fatalf("expected crossing alert: %+v", run)
	}
	res, err = http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("UI HTTP %d", res.StatusCode)
	}
	res.Body.Close()
}
