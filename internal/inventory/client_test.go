package inventory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientRetries503AndParsesProducts(t *testing.T) {
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) < 3 {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `{"products":[{"sku":"A","name":"Coffee","quantity":10,"price_cents":990}]}`)
	}))
	defer srv.Close()
	client := SupplierClient{URL: srv.URL, Attempts: 3}
	products, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count.Load() != 3 || len(products) != 1 || products[0].SKU != "A" {
		t.Fatalf("retry response incorrect: count=%d products=%+v", count.Load(), products)
	}
}

func TestClientDoesNotRetryBadRequest(t *testing.T) {
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { count.Add(1); w.WriteHeader(400) }))
	defer srv.Close()
	_, err := (SupplierClient{URL: srv.URL, Attempts: 3}).Fetch(context.Background())
	if err == nil || count.Load() != 1 {
		t.Fatalf("want single 400 request, got %d %v", count.Load(), err)
	}
}

func TestClientRejectsMissingProducts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"message":"not products"}`) }))
	defer srv.Close()
	_, err := (SupplierClient{URL: srv.URL}).Fetch(context.Background())
	if err == nil {
		t.Fatal("expected missing products to fail")
	}
}
