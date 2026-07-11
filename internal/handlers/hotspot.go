package handlers

import (
	"bufio"
	"encoding/csv"
	"net/http"
	"strings"

	"mikrotik-controller/internal/routeros"
)

// HotspotPage renders the hotspot management page
func (h *Handler) HotspotPage(w http.ResponseWriter, r *http.Request) {
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
		"Title":          "Hotspot Management",
		"Page":           "hotspot",
		"Routers":        routers,
		"SelectedRouter": selectedRouter,
	}
	h.renderPage(w, "hotspot.html", data)
}

// GetHotspotUsers returns hotspot users for a router
func (h *Handler) GetHotspotUsers(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	users, err := h.connMgr.GetHotspotUsers(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, users)
}

// AddHotspotUser adds a new hotspot user
func (h *Handler) AddHotspotUser(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	user := routeros.HotspotUser{
		Name:        r.FormValue("name"),
		Password:    r.FormValue("password"),
		Profile:     r.FormValue("profile"),
		Comment:     r.FormValue("comment"),
		Disabled:    r.FormValue("disabled") == "true",
		LimitUptime: r.FormValue("limit_uptime"),
		LimitBytes:  r.FormValue("limit_bytes"),
	}

	if user.Name == "" || user.Password == "" {
		h.errorResponse(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	if user.Profile == "" {
		user.Profile = "default"
	}

	if err := h.connMgr.AddHotspotUser(routerID, user); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]string{"message": "Hotspot user added"})
}

// UpdateHotspotUser updates an existing hotspot user
func (h *Handler) UpdateHotspotUser(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	oldName := r.PathValue("name")
	if oldName == "" {
		h.errorResponse(w, http.StatusBadRequest, "Username required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	user := routeros.HotspotUser{
		Name:        r.FormValue("name"),
		Password:    r.FormValue("password"),
		Profile:     r.FormValue("profile"),
		Comment:     r.FormValue("comment"),
		Disabled:    r.FormValue("disabled") == "true",
		LimitUptime: r.FormValue("limit_uptime"),
		LimitBytes:  r.FormValue("limit_bytes"),
	}

	if user.Profile == "" {
		user.Profile = "default"
	}

	if err := h.connMgr.UpdateHotspotUser(routerID, oldName, user); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Hotspot user updated"})
}

// DeleteHotspotUser removes a hotspot user
func (h *Handler) DeleteHotspotUser(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		h.errorResponse(w, http.StatusBadRequest, "Username required")
		return
	}

	if err := h.connMgr.DeleteHotspotUser(routerID, name); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Hotspot user deleted"})
}

// ImportHotspotUsers imports users from a CSV file
// CSV format: name,password,profile,comment
func (h *Handler) ImportHotspotUsers(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "File required")
		return
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	records, err := reader.ReadAll()
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid CSV format")
		return
	}

	imported := 0
	errors := []string{}

	for i, record := range records {
		if i == 0 && (strings.ToLower(record[0]) == "name" || strings.ToLower(record[0]) == "username") {
			continue
		}

		if len(record) < 2 {
			errors = append(errors, "Row "+intToStr(i+1)+": insufficient columns")
			continue
		}

		user := routeros.HotspotUser{
			Name:     strings.TrimSpace(record[0]),
			Password: strings.TrimSpace(record[1]),
			Profile:  "default",
		}

		if len(record) > 2 && record[2] != "" {
			user.Profile = strings.TrimSpace(record[2])
		}
		if len(record) > 3 {
			user.Comment = strings.TrimSpace(record[3])
		}

		if err := h.connMgr.AddHotspotUser(routerID, user); err != nil {
			errors = append(errors, "Row "+intToStr(i+1)+": "+err.Error())
			continue
		}
		imported++
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"imported": imported,
		"errors":   errors,
		"total":    len(records),
	})
}

// GetHotspotProfiles returns hotspot user profiles
func (h *Handler) GetHotspotProfiles(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	profiles, err := h.connMgr.GetHotspotProfiles(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, profiles)
}

// AddHotspotProfile creates a new hotspot user profile
func (h *Handler) AddHotspotProfile(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	profile := routeros.HotspotProfile{
		Name:           r.FormValue("name"),
		RateLimit:      r.FormValue("rate_limit"),
		SessionTimeout: r.FormValue("session_timeout"),
		IdleTimeout:    r.FormValue("idle_timeout"),
		SharedUsers:    r.FormValue("shared_users"),
		LoginBy:        r.FormValue("login_by"),
	}

	if profile.Name == "" {
		h.errorResponse(w, http.StatusBadRequest, "Profile name is required")
		return
	}

	if err := h.connMgr.AddHotspotProfile(routerID, profile); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]string{"message": "Profile created"})
}

// GetHotspotActive returns active hotspot sessions
func (h *Handler) GetHotspotActive(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	sessions, err := h.connMgr.GetHotspotActive(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, sessions)
}

// GetHotspotServers returns hotspot server configurations
func (h *Handler) GetHotspotServers(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	servers, err := h.connMgr.GetHotspotServers(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, servers)
}
