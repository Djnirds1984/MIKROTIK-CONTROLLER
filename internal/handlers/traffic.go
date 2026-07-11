package handlers

import (
	"database/sql"
	"net/http"
)

// TrafficPage renders the traffic monitoring page
func (h *Handler) TrafficPage(w http.ResponseWriter, r *http.Request) {
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
		"Title":          "Traffic Monitor",
		"Page":           "traffic",
		"Routers":        routers,
		"SelectedRouter": selectedRouter,
	}
	h.renderPage(w, "traffic.html", data)
}

// GetTraffic returns real-time traffic data
func (h *Handler) GetTraffic(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	traffic, err := h.connMgr.GetTraffic(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, traffic)
}

// GetTrafficClients returns connected clients
func (h *Handler) GetTrafficClients(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	clients, err := h.connMgr.GetClients(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, clients)
}

// GetTrafficHistory returns historical traffic data
func (h *Handler) GetTrafficHistory(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	// Get last 60 records (30 minutes at 30s intervals)
	rows, err := h.db.Query(`
		SELECT interface_name, rx_rate, tx_rate, recorded_at 
		FROM traffic_history 
		WHERE router_id = $1 
		ORDER BY recorded_at DESC 
		LIMIT 60`,
		routerID,
	)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type historyRecord struct {
		Interface string `json:"interface"`
		RxRate    int64  `json:"rx_rate"`
		TxRate    int64  `json:"tx_rate"`
		Time      string `json:"time"`
	}

	var history []historyRecord
	for rows.Next() {
		var rec historyRecord
		if err := rows.Scan(&rec.Interface, &rec.RxRate, &rec.TxRate, &rec.Time); err != nil {
			continue
		}
		history = append(history, rec)
	}

	// Handle case where no history exists yet
	if history == nil {
		history = []historyRecord{}
	}

	h.jsonResponse(w, http.StatusOK, history)
}

// Helper to scan a single row
func scanTrafficRow(row *sql.Row) (string, int64, int64, string, error) {
	var iface string
	var rxRate, txRate int64
	var recordedAt string
	err := row.Scan(&iface, &rxRate, &txRate, &recordedAt)
	return iface, rxRate, txRate, recordedAt, err
}
