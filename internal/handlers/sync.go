package handlers

import (
	"log"

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
