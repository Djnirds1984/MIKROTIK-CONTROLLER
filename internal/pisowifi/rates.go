package pisowifi

import (
	"database/sql"
	"fmt"
	"time"
)

// Rate represents a coin-to-time mapping tier
type Rate struct {
	ID          int       `json:"id"`
	RouterID    int       `json:"router_id"`
	CoinValue   int       `json:"coin_value"`
	TimeSeconds int       `json:"time_seconds"`
	Label       string    `json:"label"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// RateStore handles rate tier CRUD operations
type RateStore struct {
	db *sql.DB
}

// NewRateStore creates a new RateStore
func NewRateStore(db *sql.DB) *RateStore {
	return &RateStore{db: db}
}

// GetRates returns all rate tiers for a router
func (rs *RateStore) GetRates(routerID int) ([]Rate, error) {
	rows, err := rs.db.Query(
		"SELECT id, router_id, coin_value, time_seconds, label, enabled, created_at FROM pisowifi_rates WHERE router_id = $1 ORDER BY coin_value",
		routerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get rates: %w", err)
	}
	defer rows.Close()

	var rates []Rate
	for rows.Next() {
		var r Rate
		if err := rows.Scan(&r.ID, &r.RouterID, &r.CoinValue, &r.TimeSeconds, &r.Label, &r.Enabled, &r.CreatedAt); err != nil {
			continue
		}
		rates = append(rates, r)
	}
	if rates == nil {
		rates = []Rate{}
	}
	return rates, nil
}

// GetRateByCoinValue finds the rate tier matching a coin value for a router
func (rs *RateStore) GetRateByCoinValue(routerID, coinValue int) (*Rate, error) {
	var r Rate
	err := rs.db.QueryRow(
		"SELECT id, router_id, coin_value, time_seconds, label, enabled, created_at FROM pisowifi_rates WHERE router_id = $1 AND coin_value = $2 AND enabled = true",
		routerID, coinValue,
	).Scan(&r.ID, &r.RouterID, &r.CoinValue, &r.TimeSeconds, &r.Label, &r.Enabled, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AddRate creates a new rate tier
func (rs *RateStore) AddRate(routerID, coinValue, timeSeconds int, label string) (*Rate, error) {
	var r Rate
	err := rs.db.QueryRow(
		`INSERT INTO pisowifi_rates (router_id, coin_value, time_seconds, label) 
		 VALUES ($1, $2, $3, $4) RETURNING id, router_id, coin_value, time_seconds, label, enabled, created_at`,
		routerID, coinValue, timeSeconds, label,
	).Scan(&r.ID, &r.RouterID, &r.CoinValue, &r.TimeSeconds, &r.Label, &r.Enabled, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("add rate: %w", err)
	}
	return &r, nil
}

// UpdateRate updates an existing rate tier
func (rs *RateStore) UpdateRate(rateID, coinValue, timeSeconds int, label string, enabled bool) error {
	_, err := rs.db.Exec(
		`UPDATE pisowifi_rates SET coin_value = $1, time_seconds = $2, label = $3, enabled = $4 WHERE id = $5`,
		coinValue, timeSeconds, label, enabled, rateID,
	)
	return err
}

// DeleteRate removes a rate tier
func (rs *RateStore) DeleteRate(rateID int) error {
	_, err := rs.db.Exec("DELETE FROM pisowifi_rates WHERE id = $1", rateID)
	return err
}
