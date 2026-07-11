package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"mikrotik-controller/internal/models"
)

// RoutersPage renders the router management page
func (h *Handler) RoutersPage(w http.ResponseWriter, r *http.Request) {
	routers, err := h.getAllRouters()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to load routers")
		return
	}

	data := map[string]interface{}{
		"Title":   "Routers",
		"Page":    "routers",
		"Routers": routers,
	}
	h.renderPage(w, "routers.html", data)
}

// AddRouter adds a new router connection
func (h *Handler) AddRouter(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	name := r.FormValue("name")
	host := r.FormValue("host")
	portStr := r.FormValue("port")
	username := r.FormValue("username")
	password := r.FormValue("password")

	if name == "" || host == "" || username == "" || password == "" {
		h.errorResponse(w, http.StatusBadRequest, "All fields are required")
		return
	}

	port := 80
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			h.errorResponse(w, http.StatusBadRequest, "Invalid port number")
			return
		}
	}

	var newID int
	err := h.db.QueryRow(
		"INSERT INTO routers (name, host, port, username, password) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		name, host, port, username, password,
	).Scan(&newID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to add router")
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"id":      newID,
		"message": "Router added successfully",
	})
}

// UpdateRouter updates an existing router
func (h *Handler) UpdateRouter(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	name := r.FormValue("name")
	host := r.FormValue("host")
	portStr := r.FormValue("port")
	username := r.FormValue("username")
	password := r.FormValue("password")

	port := 8728
	if portStr != "" {
		port, _ = strconv.Atoi(portStr)
	}

	query := "UPDATE routers SET name=$1, host=$2, port=$3, username=$4, updated_at=CURRENT_TIMESTAMP"
	args := []interface{}{name, host, port, username}

	if password != "" {
		query += ", password=$5"
		args = append(args, password)
	}

	query += fmt.Sprintf(" WHERE id=$%d", len(args)+1)
	args = append(args, routerID)

	_, err = h.db.Exec(query, args...)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to update router")
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Router updated successfully"})
}

// DeleteRouter removes a router
func (h *Handler) DeleteRouter(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	// Disconnect first
	h.connMgr.Disconnect(routerID)

	_, err = h.db.Exec("DELETE FROM routers WHERE id = $1", routerID)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to delete router")
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Router deleted successfully"})
}

// ConnectRouter connects to a router via RouterOS API
func (h *Handler) ConnectRouter(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	// Get router credentials from DB
	var router models.Router
	err = h.db.QueryRow(
		"SELECT id, name, host, port, username, password FROM routers WHERE id = $1",
		routerID,
	).Scan(&router.ID, &router.Name, &router.Host, &router.Port, &router.Username, &router.Password)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, "Router not found")
		return
	}

	if err := h.connMgr.Connect(router); err != nil {
		h.errorResponse(w, http.StatusBadGateway, "Connection failed: "+err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Connected successfully"})
}

// DisconnectRouter disconnects from a router
func (h *Handler) DisconnectRouter(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	h.connMgr.Disconnect(routerID)
	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Disconnected successfully"})
}
