package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
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

	// Captive portal params passed from MikroTik redirect
	mac := r.URL.Query().Get("mac")
	ip := r.URL.Query().Get("ip")
	dst := r.URL.Query().Get("dst")

	// Get router host for auto-login redirect back to MikroTik
	routerHost := ""
	routers, err := h.getAllRouters()
	if err == nil {
		for _, r := range routers {
			if id, ok := r["id"].(int); ok && id == routerID {
				if host, ok := r["host"].(string); ok {
					routerHost = host
				}
				break
			}
		}
	}

	data := map[string]interface{}{
		"RouterID":   routerID,
		"RouterHost": routerHost,
		"Rates":      rates,
		"MAC":        mac,
		"IP":         ip,
		"Dst":        dst,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check DB for custom portal template first
	tmplStore := pisowifi.NewPortalTemplateStore(h.db)
	customTmpl, err := tmplStore.GetTemplate(routerID)
	if err == nil && customTmpl != nil && customTmpl.HTML != "" {
		// Use custom template from DB
		tmpl, err := template.New("portal").Parse(customTmpl.HTML)
		if err == nil {
			renderTemplate(w, tmpl, "", data, routerID)
			return
		}
		// Fall through to default if custom template has errors
		log.Printf("[Portal] Custom template parse failed for router %d, using default: %v", routerID, err)
	}

	// Render default embedded portal template
	tmpl, ok := h.templates["portal.html"]
	if !ok {
		http.Error(w, "Portal template not found", http.StatusInternalServerError)
		return
	}
	renderTemplate(w, tmpl, "portal.html", data, routerID)
}

// renderTemplate renders a standalone template off-screen so a rendering error
// can still be answered with a clean 500. Writing directly to the
// ResponseWriter would already have sent the 200 status and partial HTML, which
// triggers "superfluous response.WriteHeader" and leaks the error into the page.
func renderTemplate(w http.ResponseWriter, tmpl *template.Template, name string, data interface{}, routerID int) {
	var buf bytes.Buffer
	var err error
	if name == "" {
		err = tmpl.Execute(&buf, data)
	} else {
		err = tmpl.ExecuteTemplate(&buf, name, data)
	}
	if err != nil {
		log.Printf("[Portal] Render failed for router %d: %v", routerID, err)
		http.Error(w, "Portal render failed", http.StatusInternalServerError)
		return
	}
	w.Write(buf.Bytes())
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
