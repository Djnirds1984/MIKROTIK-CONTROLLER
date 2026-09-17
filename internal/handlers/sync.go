package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"mikrotik-controller/internal/routeros"
)

// SyncCreateHotspotUser creates a MikroTik hotspot user for a PisoWiFi session.
// Called after a voucher is redeemed and a new session is created.
// Non-fatal: logs errors but doesn't fail the operation.
func (h *Handler) SyncCreateHotspotUser(routerID int, username string, remainingSeconds int) {
	if !h.connMgr.IsConnected(routerID) {
		log.Printf("[PisoWiFi] Router %d not connected, skipping hotspot sync", routerID)
		return
	}

	user := routeros.HotspotUser{
		Name:        username,
		Password:    username,
		Profile:     "default",
		LimitUptime: routeros.FormatUptime(remainingSeconds),
		Comment:     "pisowifi",
	}

	if err := h.connMgr.AddHotspotUser(routerID, user); err != nil {
		log.Printf("[PisoWiFi] Failed to create hotspot user %s on router %d: %v", username, routerID, err)
	} else {
		log.Printf("[PisoWiFi] Created hotspot user %s with limit-uptime=%s on router %d", username, user.LimitUptime, routerID)
	}
}

// SyncUpdateHotspotUserTime updates the MikroTik hotspot user's limit-uptime
// to reflect the new total remaining time after a coin insertion.
// Non-fatal: logs errors but doesn't fail the operation.
func (h *Handler) SyncUpdateHotspotUserTime(routerID int, username string, remainingSeconds int) {
	if !h.connMgr.IsConnected(routerID) {
		log.Printf("[PisoWiFi] Router %d not connected, skipping hotspot sync", routerID)
		return
	}

	user := routeros.HotspotUser{
		Name:        username,
		Password:    username,
		Profile:     "default",
		LimitUptime: routeros.FormatUptime(remainingSeconds),
		Comment:     "pisowifi",
	}

	if err := h.connMgr.UpdateHotspotUser(routerID, username, user); err != nil {
		log.Printf("[PisoWiFi] Failed to update hotspot user %s on router %d: %v", username, routerID, err)
	} else {
		log.Printf("[PisoWiFi] Updated hotspot user %s limit-uptime=%s on router %d", username, user.LimitUptime, routerID)
	}
}

// SyncDisableHotspotUser disables a MikroTik hotspot user when a session expires
// or is disconnected. Non-fatal: logs errors but doesn't fail the operation.
func (h *Handler) SyncDisableHotspotUser(routerID int, username string) {
	if !h.connMgr.IsConnected(routerID) {
		log.Printf("[PisoWiFi] Router %d not connected, skipping hotspot sync", routerID)
		return
	}

	user := routeros.HotspotUser{
		Name:     username,
		Password: username,
		Profile:  "default",
		Disabled: true,
		Comment:  "pisowifi",
	}

	if err := h.connMgr.UpdateHotspotUser(routerID, username, user); err != nil {
		log.Printf("[PisoWiFi] Failed to disable hotspot user %s on router %d: %v", username, routerID, err)
	} else {
		log.Printf("[PisoWiFi] Disabled hotspot user %s on router %d", username, routerID)
	}
}

// SetupHotspotRedirect configures MikroTik to redirect all HTTP traffic to the SBC portal
// via walled garden + NAT dst-nat rule.
func (h *Handler) SetupHotspotRedirect(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	var req struct {
		SbcIP   string `json:"sbc_ip"`
		SbcPort int    `json:"sbc_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.SbcIP == "" {
		h.errorResponse(w, http.StatusBadRequest, "sbc_ip is required")
		return
	}
	if req.SbcPort == 0 {
		req.SbcPort = 8080
	}

	if err := h.connMgr.SetupHotspotRedirect(routerID, req.SbcIP, req.SbcPort); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to setup redirect: "+err.Error())
		return
	}

	log.Printf("[PisoWiFi] Setup hotspot redirect on router %d to %s:%d", routerID, req.SbcIP, req.SbcPort)
	h.jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Hotspot redirect configured successfully",
	})
}

// RemoveHotspotRedirect removes the pisowifi redirect rules from MikroTik
func (h *Handler) RemoveHotspotRedirect(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := h.connMgr.RemoveHotspotRedirect(routerID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to remove redirect: "+err.Error())
		return
	}

	log.Printf("[PisoWiFi] Removed hotspot redirect on router %d", routerID)
	h.jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Hotspot redirect removed successfully",
	})
}

// CheckHotspotSetup returns the PisoWiFi setup status for a router
func (h *Handler) CheckHotspotSetup(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if !h.connMgr.IsConnected(routerID) {
		h.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"connected":        false,
			"fully_configured": false,
			"message":          "Router not connected",
		})
		return
	}

	status, err := h.connMgr.CheckHotspotSetup(routerID)
	if err != nil {
		h.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"connected":        true,
			"fully_configured": false,
			"error":            err.Error(),
		})
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"connected":          true,
		"hotspot_running":    status.HotspotRunning,
		"walled_garden":      status.WalledGarden,
		"nat_redirect":       status.NatRedirect,
		"fully_configured":   status.FullyConfigured,
		"walled_garden_host": status.WalledGardenHost,
	})
}
