package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"mikrotik-controller/internal/pisowifi"
)

// PisoWifiPage renders the PisoWiFi admin dashboard
func (h *Handler) PisoWifiPage(w http.ResponseWriter, r *http.Request) {
	routers, err := h.getAllRouters()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to load routers")
		return
	}

	selectedRouter := 0
	if r.URL.Query().Get("router") != "" {
		selectedRouter, _ = parseInt(r.URL.Query().Get("router"))
	}

	data := map[string]interface{}{
		"Title":          "PisoWiFi",
		"Page":           "pisowifi",
		"Routers":        routers,
		"SelectedRouter": selectedRouter,
	}
	h.renderPage(w, "pisowifi.html", data)
}

// --- Rates API ---

func (h *Handler) GetPisoRates(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	store := pisowifi.NewRateStore(h.db)
	rates, err := store.GetRates(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.jsonResponse(w, http.StatusOK, rates)
}

func (h *Handler) AddPisoRate(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	coinValue, _ := strconv.Atoi(r.FormValue("coin_value"))
	timeSeconds, _ := strconv.Atoi(r.FormValue("time_seconds"))
	label := r.FormValue("label")

	if coinValue <= 0 || timeSeconds <= 0 {
		h.errorResponse(w, http.StatusBadRequest, "coin_value and time_seconds must be positive")
		return
	}

	store := pisowifi.NewRateStore(h.db)
	rate, err := store.AddRate(routerID, coinValue, timeSeconds, label)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, rate)
}

func (h *Handler) UpdatePisoRate(w http.ResponseWriter, r *http.Request) {
	rateIDStr := r.PathValue("rateId")
	rateID, _ := strconv.Atoi(rateIDStr)

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	coinValue, _ := strconv.Atoi(r.FormValue("coin_value"))
	timeSeconds, _ := strconv.Atoi(r.FormValue("time_seconds"))
	label := r.FormValue("label")
	enabled := r.FormValue("enabled") == "true"

	store := pisowifi.NewRateStore(h.db)
	if err := store.UpdateRate(rateID, coinValue, timeSeconds, label, enabled); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Rate updated"})
}

func (h *Handler) DeletePisoRate(w http.ResponseWriter, r *http.Request) {
	rateIDStr := r.PathValue("rateId")
	rateID, _ := strconv.Atoi(rateIDStr)

	store := pisowifi.NewRateStore(h.db)
	if err := store.DeleteRate(rateID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Rate deleted"})
}

// --- Sessions API ---

func (h *Handler) GetPisoSessions(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	store := pisowifi.NewSessionStore(h.db)
	sessions, err := store.GetActiveSessions(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, sessions)
}

func (h *Handler) PausePisoSession(w http.ResponseWriter, r *http.Request) {
	sidStr := r.PathValue("sid")
	sid, _ := strconv.Atoi(sidStr)

	store := pisowifi.NewSessionStore(h.db)
	if err := store.PauseSession(sid); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Session paused"})
}

func (h *Handler) ResumePisoSession(w http.ResponseWriter, r *http.Request) {
	sidStr := r.PathValue("sid")
	sid, _ := strconv.Atoi(sidStr)

	store := pisowifi.NewSessionStore(h.db)
	if err := store.ResumeSession(sid); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Session resumed"})
}

func (h *Handler) DisconnectPisoSession(w http.ResponseWriter, r *http.Request) {
	sidStr := r.PathValue("sid")
	sid, _ := strconv.Atoi(sidStr)

	store := pisowifi.NewSessionStore(h.db)

	// Get session before disconnecting to know router_id and username
	session, err := store.GetSession(sid)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, "Session not found")
		return
	}

	if err := store.DisconnectSession(sid); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Sync to MikroTik: disable hotspot user
	h.SyncDisableHotspotUser(session.RouterID, session.Username)

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Session disconnected"})
}

func (h *Handler) ExtendPisoSession(w http.ResponseWriter, r *http.Request) {
	sidStr := r.PathValue("sid")
	sid, _ := strconv.Atoi(sidStr)

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	seconds, _ := strconv.Atoi(r.FormValue("seconds"))
	if seconds <= 0 {
		h.errorResponse(w, http.StatusBadRequest, "seconds must be positive")
		return
	}

	store := pisowifi.NewSessionStore(h.db)
	if err := store.AddTime(sid, seconds); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Sync to MikroTik: update hotspot user's limit-uptime
	session, sessionErr := store.GetSession(sid)
	if sessionErr == nil {
		h.SyncUpdateHotspotUserTime(session.RouterID, session.Username, session.RemainingSeconds)
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Added %d seconds", seconds)})
}

// --- Vouchers API ---

func (h *Handler) GetPisoVouchers(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	store := pisowifi.NewVoucherStore(h.db)
	vouchers, err := store.GetVouchers(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, vouchers)
}

func (h *Handler) GeneratePisoVoucher(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	timeSeconds, _ := strconv.Atoi(r.FormValue("time_seconds"))
	if timeSeconds <= 0 {
		h.errorResponse(w, http.StatusBadRequest, "time_seconds must be positive")
		return
	}

	store := pisowifi.NewVoucherStore(h.db)
	v, err := store.CreateVoucher(routerID, timeSeconds)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, v)
}

func (h *Handler) BatchGeneratePisoVouchers(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	timeSeconds, _ := strconv.Atoi(r.FormValue("time_seconds"))
	count, _ := strconv.Atoi(r.FormValue("count"))

	if timeSeconds <= 0 || count <= 0 {
		h.errorResponse(w, http.StatusBadRequest, "time_seconds and count must be positive")
		return
	}

	store := pisowifi.NewVoucherStore(h.db)
	vouchers, err := store.BatchGenerate(routerID, timeSeconds, count)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"vouchers": vouchers,
		"count":    len(vouchers),
	})
}

func (h *Handler) ExportPisoVouchers(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	store := pisowifi.NewVoucherStore(h.db)
	vouchers, err := store.GetAvailableVouchers(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=vouchers.csv")

	writer := csv.NewWriter(w)
	writer.Write([]string{"Code", "Time (seconds)", "Status", "Created"})
	for _, v := range vouchers {
		writer.Write([]string{v.Code, strconv.Itoa(v.TimeSeconds), v.Status, v.CreatedAt.Format("2006-01-02 15:04")})
	}
	writer.Flush()
}

// --- Earnings API ---

func (h *Handler) GetPisoEarnings(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	coinStore := pisowifi.NewCoinStore(h.db)
	summary, err := coinStore.GetEarningsSummary(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coinLog, _ := coinStore.GetCoinLog(routerID, 50)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"summary": summary,
		"recent":  coinLog,
	})
}

// --- Devices API ---

func (h *Handler) GetPisoDevices(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	coinStore := pisowifi.NewCoinStore(h.db)
	devices, err := coinStore.GetDevices(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, devices)
}

func (h *Handler) RegisterPisoDevice(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	deviceID := r.FormValue("device_id")
	name := r.FormValue("name")
	apiKey := r.FormValue("api_key")

	if deviceID == "" {
		h.errorResponse(w, http.StatusBadRequest, "device_id is required")
		return
	}

	coinStore := pisowifi.NewCoinStore(h.db)
	device, err := coinStore.RegisterDevice(routerID, deviceID, name, apiKey)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, device)
}

func (h *Handler) DeletePisoDevice(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	devID := r.PathValue("devId")
	coinStore := pisowifi.NewCoinStore(h.db)
	if err := coinStore.DeleteDevice(routerID, devID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Device removed"})
}

// --- Portal Template Editor API ---

// GetPortalTemplate returns the current portal HTML template for editing
func (h *Handler) GetPortalTemplate(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	store := pisowifi.NewPortalTemplateStore(h.db)
	tmpl, err := store.GetTemplate(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if tmpl == nil {
		// Return the default embedded template
		defaultHTML := h.getDefaultPortalHTML()
		h.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"html":      defaultHTML,
			"is_custom": false,
		})
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"html":       tmpl.HTML,
		"is_custom":  true,
		"updated_at": tmpl.UpdatedAt,
	})
}

// SavePortalTemplate saves a custom portal HTML template
func (h *Handler) SavePortalTemplate(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	var req struct {
		HTML string `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.HTML == "" {
		h.errorResponse(w, http.StatusBadRequest, "HTML content is required")
		return
	}

	// Validate that the template parses correctly
	_, err = template.New("preview").Parse(req.HTML)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Template syntax error: "+err.Error())
		return
	}

	store := pisowifi.NewPortalTemplateStore(h.db)
	tmpl, err := store.SaveTemplate(routerID, req.HTML)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Invalidate the cached portal template so the next request uses the new one
	h.invalidatePortalCache(routerID)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message":    "Portal template saved",
		"updated_at": tmpl.UpdatedAt,
	})
}

// ResetPortalTemplate removes the custom template and reverts to default
func (h *Handler) ResetPortalTemplate(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	store := pisowifi.NewPortalTemplateStore(h.db)
	if err := store.ResetTemplate(routerID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.invalidatePortalCache(routerID)
	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Portal template reset to default"})
}

// PreviewPortal renders the portal with the provided HTML for live preview
func (h *Handler) PreviewPortal(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	// Read the HTML from the request body (for unsaved preview)
	var htmlContent string
	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.errorResponse(w, http.StatusBadRequest, "Failed to read body")
			return
		}
		htmlContent = string(body)
	} else {
		// GET: use saved template or default
		store := pisowifi.NewPortalTemplateStore(h.db)
		tmpl, _ := store.GetTemplate(routerID)
		if tmpl != nil {
			htmlContent = tmpl.HTML
		} else {
			htmlContent = h.getDefaultPortalHTML()
		}
	}

	// Get rates and router info for template data
	rateStore := pisowifi.NewRateStore(h.db)
	rates, _ := rateStore.GetRates(routerID)

	routerHost := ""
	routers, _ := h.getAllRouters()
	for _, rt := range routers {
		if id, ok := rt["id"].(int); ok && id == routerID {
			if host, ok := rt["host"].(string); ok {
				routerHost = host
			}
			break
		}
	}

	data := map[string]interface{}{
		"RouterID":   routerID,
		"RouterHost": routerHost,
		"Rates":      rates,
		"MAC":        "",
		"IP":         "",
		"Dst":        "",
	}

	tmpl, err := template.New("preview").Parse(htmlContent)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// getDefaultPortalHTML returns the default embedded portal template HTML
func (h *Handler) getDefaultPortalHTML() string {
	if h.DefaultPortalHTML != "" {
		return h.DefaultPortalHTML
	}
	return "<!-- portal template not available -->"
}

// invalidatePortalCache clears any cached portal template for a router
func (h *Handler) invalidatePortalCache(routerID int) {
	// Currently templates are loaded fresh each request, no cache to invalidate
	// This is a placeholder for future caching
}
