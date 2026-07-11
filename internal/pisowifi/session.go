package pisowifi

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Session represents an active PisoWiFi session
type Session struct {
	ID               int        `json:"id"`
	RouterID         int        `json:"router_id"`
	Username         string     `json:"username"`
	MACAddress       string     `json:"mac_address"`
	IPAddress        string     `json:"ip_address"`
	TotalSeconds     int        `json:"total_seconds"`
	UsedSeconds      int        `json:"used_seconds"`
	RemainingSeconds int        `json:"remaining_seconds"`
	Status           string     `json:"status"`
	PausedAt         *time.Time `json:"paused_at"`
	StartedAt        time.Time  `json:"started_at"`
	ExpiresAt        *time.Time `json:"expires_at"`
}

// SessionStore handles session DB operations
type SessionStore struct {
	db *sql.DB
}

// NewSessionStore creates a new SessionStore
func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

// CreateSession creates a new session
func (ss *SessionStore) CreateSession(routerID int, username, macAddr, ipAddr string, totalSeconds int) (*Session, error) {
	var s Session
	err := ss.db.QueryRow(
		`INSERT INTO pisowifi_sessions (router_id, username, mac_address, ip_address, total_seconds, remaining_seconds, status)
		 VALUES ($1, $2, $3, $4, $5, $5, 'active')
		 RETURNING id, router_id, username, mac_address, ip_address, total_seconds, used_seconds, remaining_seconds, status, paused_at, started_at, expires_at`,
		routerID, username, macAddr, ipAddr, totalSeconds,
	).Scan(&s.ID, &s.RouterID, &s.Username, &s.MACAddress, &s.IPAddress, &s.TotalSeconds, &s.UsedSeconds, &s.RemainingSeconds, &s.Status, &s.PausedAt, &s.StartedAt, &s.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &s, nil
}

// GetSession retrieves a session by ID
func (ss *SessionStore) GetSession(sessionID int) (*Session, error) {
	var s Session
	err := ss.db.QueryRow(
		`SELECT id, router_id, username, mac_address, ip_address, total_seconds, used_seconds, remaining_seconds, status, paused_at, started_at, expires_at
		 FROM pisowifi_sessions WHERE id = $1`,
		sessionID,
	).Scan(&s.ID, &s.RouterID, &s.Username, &s.MACAddress, &s.IPAddress, &s.TotalSeconds, &s.UsedSeconds, &s.RemainingSeconds, &s.Status, &s.PausedAt, &s.StartedAt, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetActiveSessions returns all active/paused sessions for a router
func (ss *SessionStore) GetActiveSessions(routerID int) ([]Session, error) {
	rows, err := ss.db.Query(
		`SELECT id, router_id, username, mac_address, ip_address, total_seconds, used_seconds, remaining_seconds, status, paused_at, started_at, expires_at
		 FROM pisowifi_sessions WHERE router_id = $1 AND status IN ('active', 'paused') ORDER BY started_at DESC`,
		routerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.RouterID, &s.Username, &s.MACAddress, &s.IPAddress, &s.TotalSeconds, &s.UsedSeconds, &s.RemainingSeconds, &s.Status, &s.PausedAt, &s.StartedAt, &s.ExpiresAt); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	if sessions == nil {
		sessions = []Session{}
	}
	return sessions, nil
}

// GetSessionByMAC finds an active session by MAC address
func (ss *SessionStore) GetSessionByMAC(routerID int, macAddr string) (*Session, error) {
	var s Session
	err := ss.db.QueryRow(
		`SELECT id, router_id, username, mac_address, ip_address, total_seconds, used_seconds, remaining_seconds, status, paused_at, started_at, expires_at
		 FROM pisowifi_sessions WHERE router_id = $1 AND mac_address = $2 AND status IN ('active', 'paused') ORDER BY started_at DESC LIMIT 1`,
		routerID, macAddr,
	).Scan(&s.ID, &s.RouterID, &s.Username, &s.MACAddress, &s.IPAddress, &s.TotalSeconds, &s.UsedSeconds, &s.RemainingSeconds, &s.Status, &s.PausedAt, &s.StartedAt, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// AddTime adds seconds to a session (coin insertion)
func (ss *SessionStore) AddTime(sessionID, seconds int) error {
	_, err := ss.db.Exec(
		`UPDATE pisowifi_sessions 
		 SET total_seconds = total_seconds + $1, remaining_seconds = remaining_seconds + $1
		 WHERE id = $2`,
		seconds, sessionID,
	)
	return err
}

// PauseSession pauses a session
func (ss *SessionStore) PauseSession(sessionID int) error {
	_, err := ss.db.Exec(
		"UPDATE pisowifi_sessions SET status = 'paused', paused_at = CURRENT_TIMESTAMP WHERE id = $1",
		sessionID,
	)
	return err
}

// ResumeSession resumes a paused session
func (ss *SessionStore) ResumeSession(sessionID int) error {
	_, err := ss.db.Exec(
		"UPDATE pisowifi_sessions SET status = 'active', paused_at = NULL WHERE id = $1",
		sessionID,
	)
	return err
}

// ExpireSession marks a session as expired
func (ss *SessionStore) ExpireSession(sessionID int) error {
	_, err := ss.db.Exec(
		"UPDATE pisowifi_sessions SET status = 'expired', expires_at = CURRENT_TIMESTAMP WHERE id = $1",
		sessionID,
	)
	return err
}

// DisconnectSession marks a session as disconnected
func (ss *SessionStore) DisconnectSession(sessionID int) error {
	_, err := ss.db.Exec(
		"UPDATE pisowifi_sessions SET status = 'disconnected', expires_at = CURRENT_TIMESTAMP WHERE id = $1",
		sessionID,
	)
	return err
}

// TickActiveSessions increments used_seconds and decrements remaining_seconds for active sessions
// Returns list of session IDs that have expired (remaining <= 0)
func (ss *SessionStore) TickActiveSessions(intervalSeconds int) ([]int, error) {
	// Update active sessions: add time used, subtract remaining
	_, err := ss.db.Exec(
		`UPDATE pisowifi_sessions 
		 SET used_seconds = used_seconds + $1, remaining_seconds = remaining_seconds - $1
		 WHERE status = 'active' AND remaining_seconds > 0`,
		intervalSeconds,
	)
	if err != nil {
		return nil, fmt.Errorf("tick sessions: %w", err)
	}

	// Find sessions that just expired (remaining went to 0 or below)
	rows, err := ss.db.Query(
		`SELECT id FROM pisowifi_sessions WHERE status = 'active' AND remaining_seconds <= 0`,
	)
	if err != nil {
		return nil, fmt.Errorf("find expired sessions: %w", err)
	}
	defer rows.Close()

	var expiredIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		expiredIDs = append(expiredIDs, id)
	}

	// Mark them as expired
	for _, id := range expiredIDs {
		ss.ExpireSession(id)
	}

	return expiredIDs, nil
}

// SessionManager runs in background to track session time
type SessionManager struct {
	db     *sql.DB
	store  *SessionStore
	stopCh chan struct{}
	// Callback for when a session expires (to disconnect from MikroTik)
	OnExpire func(session Session)
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *sql.DB) *SessionManager {
	return &SessionManager{
		db:     db,
		store:  NewSessionStore(db),
		stopCh: make(chan struct{}),
	}
}

// Start begins the background session ticker (every 5 seconds)
func (sm *SessionManager) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				expired, err := sm.store.TickActiveSessions(5)
				if err != nil {
					log.Printf("SessionManager tick error: %v", err)
					continue
				}
				for _, id := range expired {
					s, err := sm.store.GetSession(id)
					if err != nil {
						continue
					}
					if sm.OnExpire != nil {
						sm.OnExpire(*s)
					}
				}
			case <-sm.stopCh:
				return
			}
		}
	}()
	log.Println("PisoWiFi SessionManager started")
}

// Stop halts the background session ticker
func (sm *SessionManager) Stop() {
	close(sm.stopCh)
}
