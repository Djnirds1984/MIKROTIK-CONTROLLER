package pisowifi

import (
	"database/sql"
	"fmt"
	"time"
)

// CoinEvent represents a coin insertion event from a NodeMCU device
type CoinEvent struct {
	SessionID int    `json:"session_id"`
	CoinValue int    `json:"coin_value"`
	DeviceID  string `json:"device_id"`
}

// CoinDevice represents a registered NodeMCU coin acceptor device
type CoinDevice struct {
	ID        int        `json:"id"`
	RouterID  int        `json:"router_id"`
	DeviceID  string     `json:"device_id"`
	Name      string     `json:"name"`
	APIKey    string     `json:"api_key"`
	LastSeen  *time.Time `json:"last_seen"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// CoinLog represents a recorded coin insertion
type CoinLog struct {
	ID        int       `json:"id"`
	RouterID  int       `json:"router_id"`
	SessionID int       `json:"session_id"`
	DeviceID  string    `json:"device_id"`
	CoinValue int       `json:"coin_value"`
	TimeAdded int       `json:"time_added"`
	CreatedAt time.Time `json:"created_at"`
}

// CoinStore handles coin-related DB operations
type CoinStore struct {
	db *sql.DB
}

// NewCoinStore creates a new CoinStore
func NewCoinStore(db *sql.DB) *CoinStore {
	return &CoinStore{db: db}
}

// ProcessCoinEvent processes a coin event: validates session, looks up rate, adds time, logs event
func (cs *CoinStore) ProcessCoinEvent(event CoinEvent, rateStore *RateStore, sessionStore *SessionStore) (*CoinLog, error) {
	// Get the session to find the router_id
	session, err := sessionStore.GetSession(event.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if session.Status != "active" && session.Status != "paused" {
		return nil, fmt.Errorf("session is not active (status: %s)", session.Status)
	}

	// Look up rate tier for this coin value on this router
	rate, err := rateStore.GetRateByCoinValue(session.RouterID, event.CoinValue)
	if err != nil {
		return nil, fmt.Errorf("no rate tier found for %d peso on router %d: %w", event.CoinValue, session.RouterID, err)
	}

	// Add time to the session
	if err := sessionStore.AddTime(event.SessionID, rate.TimeSeconds); err != nil {
		return nil, fmt.Errorf("add time to session: %w", err)
	}

	// Log the coin event
	var coinLog CoinLog
	err = cs.db.QueryRow(
		`INSERT INTO pisowifi_coin_log (router_id, session_id, device_id, coin_value, time_added)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, router_id, session_id, device_id, coin_value, time_added, created_at`,
		session.RouterID, event.SessionID, event.DeviceID, event.CoinValue, rate.TimeSeconds,
	).Scan(&coinLog.ID, &coinLog.RouterID, &coinLog.SessionID, &coinLog.DeviceID, &coinLog.CoinValue, &coinLog.TimeAdded, &coinLog.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("log coin event: %w", err)
	}

	return &coinLog, nil
}

// GetCoinLog returns coin log entries for a router
func (cs *CoinStore) GetCoinLog(routerID int, limit int) ([]CoinLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := cs.db.Query(
		`SELECT id, router_id, session_id, device_id, coin_value, time_added, created_at
		 FROM pisowifi_coin_log WHERE router_id = $1 ORDER BY created_at DESC LIMIT $2`,
		routerID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []CoinLog
	for rows.Next() {
		var l CoinLog
		if err := rows.Scan(&l.ID, &l.RouterID, &l.SessionID, &l.DeviceID, &l.CoinValue, &l.TimeAdded, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []CoinLog{}
	}
	return logs, nil
}

// GetEarningsSummary returns total earnings for a router grouped by time period
func (cs *CoinStore) GetEarningsSummary(routerID int) (map[string]int, error) {
	summary := map[string]int{
		"today":      0,
		"this_week":  0,
		"this_month": 0,
		"total":      0,
	}

	rows, err := cs.db.Query(
		`SELECT coin_value, created_at FROM pisowifi_coin_log WHERE router_id = $1 ORDER BY created_at DESC`,
		routerID,
	)
	if err != nil {
		return summary, err
	}
	defer rows.Close()

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(now.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	for rows.Next() {
		var coinValue int
		var createdAt time.Time
		if err := rows.Scan(&coinValue, &createdAt); err != nil {
			continue
		}
		summary["total"] += coinValue
		if createdAt.After(monthStart) {
			summary["this_month"] += coinValue
		}
		if createdAt.After(weekStart) {
			summary["this_week"] += coinValue
		}
		if createdAt.After(todayStart) {
			summary["today"] += coinValue
		}
	}

	return summary, nil
}

// --- Device Registry ---

// RegisterDevice registers a new NodeMCU device
func (cs *CoinStore) RegisterDevice(routerID int, deviceID, name, apiKey string) (*CoinDevice, error) {
	var d CoinDevice
	err := cs.db.QueryRow(
		`INSERT INTO pisowifi_devices (router_id, device_id, name, api_key)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, router_id, device_id, name, api_key, last_seen, status, created_at`,
		routerID, deviceID, name, apiKey,
	).Scan(&d.ID, &d.RouterID, &d.DeviceID, &d.Name, &d.APIKey, &d.LastSeen, &d.Status, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("register device: %w", err)
	}
	return &d, nil
}

// GetDevices returns all registered devices for a router
func (cs *CoinStore) GetDevices(routerID int) ([]CoinDevice, error) {
	rows, err := cs.db.Query(
		`SELECT id, router_id, device_id, name, api_key, last_seen, status, created_at
		 FROM pisowifi_devices WHERE router_id = $1 ORDER BY device_id`,
		routerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []CoinDevice
	for rows.Next() {
		var d CoinDevice
		if err := rows.Scan(&d.ID, &d.RouterID, &d.DeviceID, &d.Name, &d.APIKey, &d.LastSeen, &d.Status, &d.CreatedAt); err != nil {
			continue
		}
		devices = append(devices, d)
	}
	if devices == nil {
		devices = []CoinDevice{}
	}
	return devices, nil
}

// GetDevice retrieves a device by device_id
func (cs *CoinStore) GetDevice(deviceID string) (*CoinDevice, error) {
	var d CoinDevice
	err := cs.db.QueryRow(
		`SELECT id, router_id, device_id, name, api_key, last_seen, status, created_at
		 FROM pisowifi_devices WHERE device_id = $1`,
		deviceID,
	).Scan(&d.ID, &d.RouterID, &d.DeviceID, &d.Name, &d.APIKey, &d.LastSeen, &d.Status, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDeviceHeartbeat updates the last_seen timestamp and status
func (cs *CoinStore) UpdateDeviceHeartbeat(deviceID string) error {
	_, err := cs.db.Exec(
		`UPDATE pisowifi_devices SET last_seen = CURRENT_TIMESTAMP, status = 'online' WHERE device_id = $1`,
		deviceID,
	)
	return err
}

// DeleteDevice removes a device
func (cs *CoinStore) DeleteDevice(routerID int, deviceID string) error {
	_, err := cs.db.Exec(
		"DELETE FROM pisowifi_devices WHERE router_id = $1 AND device_id = $2",
		routerID, deviceID,
	)
	return err
}
