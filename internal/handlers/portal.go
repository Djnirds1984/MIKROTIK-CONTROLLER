package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"mikrotik-controller/internal/pisowifi"
)

// PortalPage renders the user-facing PisoWiFi portal
func (h *Handler) PortalPage(w http.ResponseWriter, r *http.Request) {
	routerIDStr := r.PathValue("router_id")
	routerID, _ := strconv.Atoi(routerIDStr)

	// Get rates for display
	rateStore := pisowifi.NewRateStore(h.db)
	rates, _ := rateStore.GetRates(routerID)

	data := map[string]interface{}{
		"RouterID": routerID,
		"Rates":    rates,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Render standalone portal template (no layout)
	tmpl, ok := h.templates["portal.html"]
	if !ok {
		http.Error(w, "Portal template not found", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

// PortalRedeem redeems a voucher code and creates a new session
func (h *Handler) PortalRedeem(w http.ResponseWriter, r *http.Request) {
	routerIDStr := r.PathValue("router_id")
	routerID, _ := strconv.Atoi(routerIDStr)

	var req struct {
		Code      string `json:"code"`
		MACAddr   string `json:"mac_address"`
		IPAddress string `json:"ip_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Code == "" {
		h.errorResponse(w, http.StatusBadRequest, "Voucher code is required")
		return
	}

	voucherStore := pisowifi.NewVoucherStore(h.db)
	sessionStore := pisowifi.NewSessionStore(h.db)

	// Validate voucher
	voucher, err := voucherStore.GetVoucher(req.Code)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, "Invalid voucher code")
		return
	}

	if voucher.Status != "available" {
		h.errorResponse(w, http.StatusBadRequest, "Voucher already used or expired")
		return
	}

	// Generate a username for the hotspot user
	username := "piso_" + voucher.Code

	// Create session
	session, err := sessionStore.CreateSession(routerID, username, req.MACAddr, req.IPAddress, voucher.TimeSeconds)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to create session: "+err.Error())
		return
	}

	// Mark voucher as used
	voucherStore.RedeemVoucher(req.Code, session.ID)

	// Sync to MikroTik: create hotspot user with limit-uptime
	h.SyncCreateHotspotUser(routerID, session.Username, session.RemainingSeconds)

	h.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Voucher redeemed successfully",
		"session": session,
		"voucher": voucher,
	})
}

// PortalSession returns the current session status
func (h *Handler) PortalSession(w http.ResponseWriter, r *http.Request) {
	routerIDStr := r.PathValue("router_id")
	routerID, _ := strconv.Atoi(routerIDStr)

	macAddr := r.URL.Query().Get("mac")
	if macAddr == "" {
		h.errorResponse(w, http.StatusBadRequest, "mac parameter required")
		return
	}

	sessionStore := pisowifi.NewSessionStore(h.db)
	session, err := sessionStore.GetSessionByMAC(routerID, macAddr)
	if err != nil {
		h.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"active":  false,
			"message": "No active session",
		})
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"active":            true,
		"session":           session,
		"remaining_seconds": session.RemainingSeconds,
		"status":            session.Status,
		"low_time":          session.RemainingSeconds > 0 && session.RemainingSeconds <= 300, // 5 min warning
	})
}

// PortalPause pauses the current session
func (h *Handler) PortalPause(w http.ResponseWriter, r *http.Request) {
	routerIDStr := r.PathValue("router_id")
	routerID, _ := strconv.Atoi(routerIDStr)

	var req struct {
		MACAddr string `json:"mac_address"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	sessionStore := pisowifi.NewSessionStore(h.db)
	session, err := sessionStore.GetSessionByMAC(routerID, req.MACAddr)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, "No active session found")
		return
	}

	if err := sessionStore.PauseSession(session.ID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Session paused"})
}

// PortalResume resumes a paused session
func (h *Handler) PortalResume(w http.ResponseWriter, r *http.Request) {
	routerIDStr := r.PathValue("router_id")
	routerID, _ := strconv.Atoi(routerIDStr)

	var req struct {
		MACAddr string `json:"mac_address"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	sessionStore := pisowifi.NewSessionStore(h.db)
	session, err := sessionStore.GetSessionByMAC(routerID, req.MACAddr)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, "No active session found")
		return
	}

	if err := sessionStore.ResumeSession(session.ID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Session resumed"})
}

// PortalLogout ends the session
func (h *Handler) PortalLogout(w http.ResponseWriter, r *http.Request) {
	routerIDStr := r.PathValue("router_id")
	routerID, _ := strconv.Atoi(routerIDStr)

	var req struct {
		MACAddr string `json:"mac_address"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	sessionStore := pisowifi.NewSessionStore(h.db)
	session, err := sessionStore.GetSessionByMAC(routerID, req.MACAddr)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, "No active session found")
		return
	}

	if err := sessionStore.DisconnectSession(session.ID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Session ended"})
}
