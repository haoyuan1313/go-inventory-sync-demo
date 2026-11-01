package inventory

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// MockSupplier provides local demo data. It is not a production API integration.
type MockSupplier struct {
	mu       sync.Mutex
	products map[string]Product
	failNext int
}

func NewMockSupplier() *MockSupplier {
	entries := []Product{
		{SKU: "COFFEE-001", Name: "House Blend Beans 1kg", Quantity: 22, PriceCents: 8900},
		{SKU: "MUG-002", Name: "Ceramic Coffee Mug", Quantity: 15, PriceCents: 3500},
		{SKU: "FILTER-003", Name: "Paper Filters", Quantity: 8, PriceCents: 1800},
		{SKU: "KETTLE-004", Name: "Pour-over Kettle", Quantity: 18, PriceCents: 12900},
	}
	p := make(map[string]Product)
	for _, x := range entries {
		p[x.SKU] = x
	}
	return &MockSupplier{products: p}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (m *MockSupplier) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/mock/supplier/items" && r.Method == http.MethodGet:
		m.mu.Lock()
		if m.failNext > 0 {
			m.failNext--
			m.mu.Unlock()
			writeJSON(w, 503, map[string]string{"error": "simulated transient upstream failure"})
			return
		}
		products := make([]Product, 0, len(m.products))
		for _, p := range m.products {
			products = append(products, p)
		}
		m.mu.Unlock()
		sort.Slice(products, func(i, j int) bool { return products[i].SKU < products[j].SKU })
		writeJSON(w, http.StatusOK, supplierResponse{Products: products})
	case strings.HasPrefix(r.URL.Path, "/mock/supplier/items/") && r.Method == http.MethodPatch:
		sku := strings.TrimPrefix(r.URL.Path, "/mock/supplier/items/")
		var body struct {
			Quantity *int `json:"quantity"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil || body.Quantity == nil || *body.Quantity < 0 {
			writeJSON(w, 400, map[string]string{"error": "quantity must be a nonnegative integer"})
			return
		}
		m.mu.Lock()
		item, ok := m.products[sku]
		if ok {
			item.Quantity = *body.Quantity
			m.products[sku] = item
		}
		m.mu.Unlock()
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "unknown SKU"})
			return
		}
		writeJSON(w, 200, item)
	case r.URL.Path == "/mock/supplier/fail-next" && r.Method == http.MethodPost:
		var body struct {
			Count int `json:"count"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil || body.Count < 0 || body.Count > 5 {
			writeJSON(w, 400, map[string]string{"error": "count must be between 0 and 5"})
			return
		}
		m.mu.Lock()
		m.failNext = body.Count
		m.mu.Unlock()
		writeJSON(w, 200, map[string]int{"fail_next": body.Count})
	default:
		w.Header().Set("Allow", "GET, PATCH, POST")
		writeJSON(w, 404, map[string]string{"error": "route not found"})
	}
}
