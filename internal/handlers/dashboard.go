package handlers

import (
	"net/http"
)

// Dashboard renders the main dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	routers, err := h.getAllRouters()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to load routers")
		return
	}

	// Get system info for connected routers
	var statuses []map[string]interface{}
	for _, router := range routers {
		id := router["id"].(int)
		if h.connMgr.IsConnected(id) {
			status, err := h.connMgr.GetSystemInfo(id)
			if err == nil {
				statuses = append(statuses, map[string]interface{}{
					"router":       router,
					"connected":    true,
					"board_name":   status.BoardName,
					"firmware":     status.FirmwareVersion,
					"uptime":       status.Uptime,
					"cpu_load":     status.CPULoad,
					"free_memory":  status.FreeMemory,
					"total_memory": status.TotalMemory,
					"temperature":  status.Temperature,
					"identity":     status.Identity,
				})
				continue
			}
		}
		statuses = append(statuses, map[string]interface{}{
			"router":    router,
			"connected": false,
		})
	}

	data := map[string]interface{}{
		"Title":   "Dashboard",
		"Page":    "dashboard",
		"Routers": statuses,
	}
	h.renderPage(w, "dashboard.html", data)
}
