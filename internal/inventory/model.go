package inventory

import (
	"fmt"
	"strings"
	"time"
)

type Product struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	Quantity   int    `json:"quantity"`
	PriceCents int    `json:"price_cents"`
}

func (p Product) Validate() error {
	if strings.TrimSpace(p.SKU) == "" || len(p.SKU) > 64 {
		return fmt.Errorf("invalid SKU")
	}
	if strings.TrimSpace(p.Name) == "" || len(p.Name) > 200 {
		return fmt.Errorf("invalid name for SKU %s", p.SKU)
	}
	if p.Quantity < 0 || p.PriceCents < 0 {
		return fmt.Errorf("negative quantity or price for SKU %s", p.SKU)
	}
	return nil
}

type Item struct {
	Product
	LastSyncedAt time.Time `json:"last_synced_at"`
}

type Alert struct {
	ID        string    `json:"id"`
	SKU       string    `json:"sku"`
	Name      string    `json:"name"`
	From      *int      `json:"from_quantity,omitempty"`
	To        int       `json:"to_quantity"`
	Threshold int       `json:"threshold"`
	CreatedAt time.Time `json:"created_at"`
}

type Run struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Fetched    int       `json:"fetched"`
	Inserted   int       `json:"inserted"`
	Updated    int       `json:"updated"`
	Unchanged  int       `json:"unchanged"`
	Alerts     int       `json:"alerts"`
	Error      string    `json:"error,omitempty"`
}

type State struct {
	Items  map[string]Item `json:"items"`
	Alerts []Alert         `json:"alerts"`
	Runs   []Run           `json:"runs"`
}

func emptyState() State { return State{Items: map[string]Item{}, Alerts: []Alert{}, Runs: []Run{}} }

func (s State) clone() State {
	c := emptyState()
	for key, item := range s.Items {
		c.Items[key] = item
	}
	c.Alerts = append(c.Alerts, s.Alerts...)
	c.Runs = append(c.Runs, s.Runs...)
	return c
}
