package handlers

import (
	"bufio"
	"encoding/csv"
	"net/http"
	"strings"

	"mikrotik-controller/internal/models"
)

// PPPoEPage renders the PPPoE management page
func (h *Handler) PPPoEPage(w http.ResponseWriter, r *http.Request) {
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
		"Title":          "PPPoE Management",
		"Page":           "pppoe",
		"Routers":        routers,
		"SelectedRouter": selectedRouter,
	}
	h.renderPage(w, "pppoe.html", data)
}

// GetPPPoEUsers returns PPPoE users for a router
func (h *Handler) GetPPPoEUsers(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	users, err := h.connMgr.GetPPPoESecrets(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, users)
}

// AddPPPoEUser adds a new PPPoE user
func (h *Handler) AddPPPoEUser(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	user := models.PPPoEUser{
		Name:     r.FormValue("name"),
		Password: r.FormValue("password"),
		Profile:  r.FormValue("profile"),
		Service:  r.FormValue("service"),
		Comment:  r.FormValue("comment"),
		Disabled: r.FormValue("disabled") == "true",
	}

	if user.Name == "" || user.Password == "" {
		h.errorResponse(w, http.StatusBadRequest, "Name and password are required")
		return
	}

	if user.Profile == "" {
		user.Profile = "default"
	}
	if user.Service == "" {
		user.Service = "pppoe"
	}

	if err := h.connMgr.AddPPPoESecret(routerID, user); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]string{"message": "PPPoE user added"})
}

// UpdatePPPoEUser updates a PPPoE user
func (h *Handler) UpdatePPPoEUser(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	oldName := r.PathValue("name")
	if oldName == "" {
		h.errorResponse(w, http.StatusBadRequest, "User name required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	user := models.PPPoEUser{
		Name:     r.FormValue("name"),
		Password: r.FormValue("password"),
		Profile:  r.FormValue("profile"),
		Service:  r.FormValue("service"),
		Comment:  r.FormValue("comment"),
		Disabled: r.FormValue("disabled") == "true",
	}

	if user.Profile == "" {
		user.Profile = "default"
	}
	if user.Service == "" {
		user.Service = "pppoe"
	}

	if err := h.connMgr.UpdatePPPoESecret(routerID, oldName, user); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "PPPoE user updated"})
}

// DeletePPPoEUser removes a PPPoE user
func (h *Handler) DeletePPPoEUser(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		h.errorResponse(w, http.StatusBadRequest, "User name required")
		return
	}

	if err := h.connMgr.DeletePPPoESecret(routerID, name); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "PPPoE user deleted"})
}

// ImportPPPoEUsers imports users from a CSV file
// CSV format: name,password,profile,comment
func (h *Handler) ImportPPPoEUsers(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
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
			continue // Skip header row
		}

		if len(record) < 2 {
			errors = append(errors, "Row "+intToStr(i+1)+": insufficient columns")
			continue
		}

		user := models.PPPoEUser{
			Name:     strings.TrimSpace(record[0]),
			Password: strings.TrimSpace(record[1]),
			Profile:  "default",
			Service:  "pppoe",
		}

		if len(record) > 2 && record[2] != "" {
			user.Profile = strings.TrimSpace(record[2])
		}
		if len(record) > 3 {
			user.Comment = strings.TrimSpace(record[3])
		}

		if err := h.connMgr.AddPPPoESecret(routerID, user); err != nil {
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

// GetPPPoEProfiles returns PPPoE profiles
func (h *Handler) GetPPPoEProfiles(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	profiles, err := h.connMgr.GetPPPoEProfiles(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, profiles)
}

// AddPPPoEProfile creates a new PPPoE profile
func (h *Handler) AddPPPoEProfile(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	profile := models.PPPoEProfile{
		Name:          r.FormValue("name"),
		LocalAddress:  r.FormValue("local_address"),
		RemoteAddress: r.FormValue("remote_address"),
		RateLimit:     r.FormValue("rate_limit"),
		ParentProfile: r.FormValue("parent_profile"),
	}

	if profile.Name == "" {
		h.errorResponse(w, http.StatusBadRequest, "Profile name is required")
		return
	}

	if err := h.connMgr.AddPPPoEProfile(routerID, profile); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]string{"message": "Profile created"})
}

// GetPPPoESessions returns active PPPoE sessions
func (h *Handler) GetPPPoESessions(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	sessions, err := h.connMgr.GetPPPoESessions(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, sessions)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	if neg {
		result = "-" + result
	}
	return result
}
