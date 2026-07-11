package handlers

import (
	"encoding/json"
	"net/http"

	"mikrotik-controller/internal/pisowifi"
)

// CoinEventRequest represents the JSON body from NodeMCU
type CoinEventRequest struct {
	SessionID int    `json:"session_id"`
	CoinValue int    `json:"coin_value"`
	DeviceID  string `json:"device_id"`
}

// HandleCoinEvent processes coin events from NodeMCU devices
func (h *Handler) HandleCoinEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	var req CoinEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.SessionID <= 0 || req.CoinValue <= 0 || req.DeviceID == "" {
		h.errorResponse(w, http.StatusBadRequest, "session_id, coin_value, and device_id are required")
		return
	}

	// Validate device is registered
	coinStore := pisowifi.NewCoinStore(h.db)
	device, err := coinStore.GetDevice(req.DeviceID)
	if err != nil {
		h.errorResponse(w, http.StatusUnauthorized, "Device not registered: "+req.DeviceID)
		return
	}

	// Optional: validate API key if set
	if device.APIKey != "" {
		key := r.Header.Get("X-API-Key")
		if key != device.APIKey {
			h.errorResponse(w, http.StatusUnauthorized, "Invalid API key")
			return
		}
	}

	// Update device heartbeat
	coinStore.UpdateDeviceHeartbeat(req.DeviceID)

	// Process the coin event
	rateStore := pisowifi.NewRateStore(h.db)
	sessionStore := pisowifi.NewSessionStore(h.db)

	event := pisowifi.CoinEvent{
		SessionID: req.SessionID,
		CoinValue: req.CoinValue,
		DeviceID:  req.DeviceID,
	}

	coinLog, err := coinStore.ProcessCoinEvent(event, rateStore, sessionStore)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sync to MikroTik: update hotspot user's limit-uptime with new remaining time
	session, sessionErr := sessionStore.GetSession(event.SessionID)
	if sessionErr == nil {
		h.SyncUpdateHotspotUserTime(session.RouterID, session.Username, session.RemainingSeconds)
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message":    "Coin accepted",
		"coin_value": coinLog.CoinValue,
		"time_added": coinLog.TimeAdded,
		"session_id": coinLog.SessionID,
	})
}
