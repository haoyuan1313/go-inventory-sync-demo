package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var sequence atomic.Uint64

func newID(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), sequence.Add(1))
}

type Engine struct {
	Source    Supplier
	Store     *Store
	Threshold int
	Logger    *slog.Logger
	mu        sync.Mutex
}

func (e *Engine) Sync(ctx context.Context) (Run, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	started := time.Now().UTC()
	result := Run{ID: newID("sync"), Status: "success", StartedAt: started}
	products, err := e.Source.Fetch(ctx)
	if err == nil {
		if len(products) > 1000 {
			err = fmt.Errorf("product limit of 1000 exceeded")
		}
	}
	if err == nil {
		seen := map[string]bool{}
		for _, product := range products {
			if vErr := product.Validate(); vErr != nil {
				err = vErr
				break
			}
			product.SKU = strings.TrimSpace(product.SKU)
			if seen[product.SKU] {
				err = fmt.Errorf("duplicate SKU in supplier response: %s", product.SKU)
				break
			}
			seen[product.SKU] = true
		}
	}
	if err == nil {
		result.Fetched = len(products)
		err = e.Store.Update(func(s *State) error {
			for _, p := range products {
				p.SKU = strings.TrimSpace(p.SKU)
				old, exists := s.Items[p.SKU]
				if exists && old.Product == p {
					result.Unchanged++
					continue
				}
				if exists {
					result.Updated++
				} else {
					result.Inserted++
				}
				if p.Quantity < e.Threshold && (!exists || old.Quantity >= e.Threshold) {
					var before *int
					if exists {
						quantity := old.Quantity
						before = &quantity
					}
					alert := Alert{ID: newID("alert"), SKU: p.SKU, Name: p.Name, From: before, To: p.Quantity, Threshold: e.Threshold, CreatedAt: time.Now().UTC()}
					s.Alerts = append([]Alert{alert}, s.Alerts...)
					result.Alerts++
				}
				s.Items[p.SKU] = Item{Product: p, LastSyncedAt: time.Now().UTC()}
			}
			if len(s.Alerts) > 100 {
				s.Alerts = s.Alerts[:100]
			}
			result.FinishedAt = time.Now().UTC()
			s.Runs = append([]Run{result}, s.Runs...)
			if len(s.Runs) > 50 {
				s.Runs = s.Runs[:50]
			}
			return nil
		})
	}
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		result.FinishedAt = time.Now().UTC()
		// Best effort audit record when initial fetch or input validation fails.
		if saveErr := e.Store.Update(func(s *State) error {
			s.Runs = append([]Run{result}, s.Runs...)
			if len(s.Runs) > 50 {
				s.Runs = s.Runs[:50]
			}
			return nil
		}); saveErr != nil && e.Logger != nil {
			e.Logger.Error("failed to persist failed sync run", "error", saveErr)
		}
	}
	if e.Logger != nil {
		e.Logger.Info("inventory sync", "run_id", result.ID, "status", result.Status, "fetched", result.Fetched, "inserted", result.Inserted, "updated", result.Updated, "unchanged", result.Unchanged, "alerts", result.Alerts, "error", result.Error)
	}
	return result, err
}
