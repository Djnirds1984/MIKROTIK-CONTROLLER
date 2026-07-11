package pisowifi

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

// Voucher represents a pre-generated access code (separate from coin slot)
type Voucher struct {
	ID          int        `json:"id"`
	RouterID    int        `json:"router_id"`
	Code        string     `json:"code"`
	TimeSeconds int        `json:"time_seconds"`
	Status      string     `json:"status"`
	SessionID   *int       `json:"session_id"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// VoucherStore handles voucher DB operations
type VoucherStore struct {
	db *sql.DB
}

// NewVoucherStore creates a new VoucherStore
func NewVoucherStore(db *sql.DB) *VoucherStore {
	return &VoucherStore{db: db}
}

// GenerateCode generates a unique voucher code in PW-XXXXXX format
func GenerateCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I,O,0,1 to avoid confusion
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "PW-" + string(b)
}

// CreateVoucher creates a single voucher with the given time value
func (vs *VoucherStore) CreateVoucher(routerID, timeSeconds int) (*Voucher, error) {
	code := GenerateCode()
	expiresAt := time.Now().Add(24 * time.Hour) // 24h expiry by default

	var v Voucher
	err := vs.db.QueryRow(
		`INSERT INTO pisowifi_vouchers (router_id, code, time_seconds, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, router_id, code, time_seconds, status, session_id, created_at, expires_at`,
		routerID, code, timeSeconds, expiresAt,
	).Scan(&v.ID, &v.RouterID, &v.Code, &v.TimeSeconds, &v.Status, &v.SessionID, &v.CreatedAt, &v.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("create voucher: %w", err)
	}
	return &v, nil
}

// BatchGenerate creates multiple vouchers at once
func (vs *VoucherStore) BatchGenerate(routerID, timeSeconds, count int) ([]Voucher, error) {
	if count <= 0 || count > 100 {
		count = 10
	}

	var vouchers []Voucher
	for i := 0; i < count; i++ {
		v, err := vs.CreateVoucher(routerID, timeSeconds)
		if err != nil {
			return vouchers, fmt.Errorf("batch generate at index %d: %w", i, err)
		}
		vouchers = append(vouchers, *v)
	}
	return vouchers, nil
}

// GetVoucher retrieves a voucher by code
func (vs *VoucherStore) GetVoucher(code string) (*Voucher, error) {
	var v Voucher
	err := vs.db.QueryRow(
		`SELECT id, router_id, code, time_seconds, status, session_id, created_at, expires_at
		 FROM pisowifi_vouchers WHERE UPPER(code) = UPPER($1)`,
		code,
	).Scan(&v.ID, &v.RouterID, &v.Code, &v.TimeSeconds, &v.Status, &v.SessionID, &v.CreatedAt, &v.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetVouchers returns all vouchers for a router
func (vs *VoucherStore) GetVouchers(routerID int) ([]Voucher, error) {
	rows, err := vs.db.Query(
		`SELECT id, router_id, code, time_seconds, status, session_id, created_at, expires_at
		 FROM pisowifi_vouchers WHERE router_id = $1 ORDER BY created_at DESC`,
		routerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vouchers []Voucher
	for rows.Next() {
		var v Voucher
		if err := rows.Scan(&v.ID, &v.RouterID, &v.Code, &v.TimeSeconds, &v.Status, &v.SessionID, &v.CreatedAt, &v.ExpiresAt); err != nil {
			continue
		}
		vouchers = append(vouchers, v)
	}
	if vouchers == nil {
		vouchers = []Voucher{}
	}
	return vouchers, nil
}

// RedeemVoucher marks a voucher as used and links it to a session
func (vs *VoucherStore) RedeemVoucher(code string, sessionID int) error {
	_, err := vs.db.Exec(
		`UPDATE pisowifi_vouchers SET status = 'used', session_id = $1 WHERE UPPER(code) = UPPER($2) AND status = 'available'`,
		sessionID, code,
	)
	return err
}

// GetAvailableVouchers returns only available (unused) vouchers for a router
func (vs *VoucherStore) GetAvailableVouchers(routerID int) ([]Voucher, error) {
	rows, err := vs.db.Query(
		`SELECT id, router_id, code, time_seconds, status, session_id, created_at, expires_at
		 FROM pisowifi_vouchers WHERE router_id = $1 AND status = 'available' ORDER BY created_at DESC`,
		routerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vouchers []Voucher
	for rows.Next() {
		var v Voucher
		if err := rows.Scan(&v.ID, &v.RouterID, &v.Code, &v.TimeSeconds, &v.Status, &v.SessionID, &v.CreatedAt, &v.ExpiresAt); err != nil {
			continue
		}
		vouchers = append(vouchers, v)
	}
	if vouchers == nil {
		vouchers = []Voucher{}
	}
	return vouchers, nil
}
