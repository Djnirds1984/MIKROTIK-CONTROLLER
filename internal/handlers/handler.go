package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"mikrotik-controller/internal/routeros"
)

type Handler struct {
	templates map[string]*template.Template
	db        *sql.DB
	connMgr   *routeros.ConnectionManager
}

func New(templates map[string]*template.Template, db *sql.DB, connMgr *routeros.ConnectionManager) *Handler {
	return &Handler{
		templates: templates,
		db:        db,
		connMgr:   connMgr,
	}
}

// renderPage renders a full page template
func (h *Handler) renderPage(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, ok := h.templates[name]
	if !ok {
		log.Printf("Template not found: %s", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Template error (%s): %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderPartial renders an HTMX partial template
func (h *Handler) renderPartial(w http.ResponseWriter, page string, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, ok := h.templates[page]
	if !ok {
		log.Printf("Template not found: %s", page)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template error (%s/%s): %v", page, name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// jsonResponse sends a JSON response
func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// errorResponse sends an error response
func (h *Handler) errorResponse(w http.ResponseWriter, status int, message string) {
	h.jsonResponse(w, status, map[string]string{"error": message})
}

// getRouterID extracts and validates the router ID from URL params
func (h *Handler) getRouterID(r *http.Request) (int, error) {
	idStr := r.PathValue("id")
	return strconv.Atoi(idStr)
}

// getRouter retrieves a router from the database
func (h *Handler) getRouter(routerID int) (routerData map[string]interface{}, err error) {
	row := h.db.QueryRow("SELECT id, name, host, port, username FROM routers WHERE id = $1", routerID)
	var id int
	var name, host, username string
	var port int
	if err := row.Scan(&id, &name, &host, &port, &username); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"host":     host,
		"port":     port,
		"username": username,
	}, nil
}

// getAllRouters retrieves all routers from the database
func (h *Handler) getAllRouters() ([]map[string]interface{}, error) {
	rows, err := h.db.Query("SELECT id, name, host, port, username, created_at FROM routers ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routers []map[string]interface{}
	for rows.Next() {
		var id, port int
		var name, host, username, createdAt string
		if err := rows.Scan(&id, &name, &host, &port, &username, &createdAt); err != nil {
			continue
		}
		routers = append(routers, map[string]interface{}{
			"id":         id,
			"name":       name,
			"host":       host,
			"port":       port,
			"username":   username,
			"created_at": createdAt,
			"connected":  h.connMgr.IsConnected(id),
		})
	}
	return routers, nil
}
