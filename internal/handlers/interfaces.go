package handlers

import (
	"net/http"
)

// InterfacesPage renders the interface management page
func (h *Handler) InterfacesPage(w http.ResponseWriter, r *http.Request) {
	routers, err := h.getAllRouters()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to load routers")
		return
	}

	// Get selected router ID from query param
	selectedRouter := 0
	if r.URL.Query().Get("router") != "" {
		selectedRouter, _ = parseInt(r.URL.Query().Get("router"))
	}

	data := map[string]interface{}{
		"Title":          "Interfaces",
		"Page":           "interfaces",
		"Routers":        routers,
		"SelectedRouter": selectedRouter,
	}
	h.renderPage(w, "interfaces.html", data)
}

// GetInterfaces returns interfaces for a router
func (h *Handler) GetInterfaces(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	interfaces, err := h.connMgr.GetInterfaces(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, interfaces)
}

// ToggleInterface enables/disables an interface
func (h *Handler) ToggleInterface(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		h.errorResponse(w, http.StatusBadRequest, "Interface name required")
		return
	}

	// Get current state to toggle
	interfaces, err := h.connMgr.GetInterfaces(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	var currentState bool
	for _, iface := range interfaces {
		if iface.Name == name {
			currentState = iface.Disabled
			break
		}
	}

	// Toggle: if currently disabled, enable it; if enabled, disable it
	if err := h.connMgr.ToggleInterface(routerID, name, currentState); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "Interface toggled"})
}

// GetIPAddresses returns IP addresses for a router
func (h *Handler) GetIPAddresses(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	addresses, err := h.connMgr.GetIPAddresses(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, addresses)
}

// AddIPAddress adds an IP address to an interface
func (h *Handler) AddIPAddress(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	address := r.FormValue("address")
	iface := r.FormValue("interface")

	if address == "" || iface == "" {
		h.errorResponse(w, http.StatusBadRequest, "Address and interface are required")
		return
	}

	if err := h.connMgr.AddIPAddress(routerID, address, iface); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]string{"message": "IP address added"})
}

// DeleteIPAddress removes an IP address
func (h *Handler) DeleteIPAddress(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	address := r.PathValue("address")
	if address == "" {
		h.errorResponse(w, http.StatusBadRequest, "Address required")
		return
	}

	if err := h.connMgr.DeleteIPAddress(routerID, address); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "IP address deleted"})
}

// GetDHCPConfig returns DHCP configuration
func (h *Handler) GetDHCPConfig(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	configs, err := h.connMgr.GetDHCPConfig(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, configs)
}

// GetDNSConfig returns DNS configuration
func (h *Handler) GetDNSConfig(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	config, err := h.connMgr.GetDNSConfig(routerID)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, config)
}

// UpdateDNS updates DNS settings
func (h *Handler) UpdateDNS(w http.ResponseWriter, r *http.Request) {
	routerID, err := h.getRouterID(r)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid router ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	servers := r.FormValue("servers")
	allowRemote := r.FormValue("allow_remote") == "true"

	if err := h.connMgr.UpdateDNS(routerID, servers, allowRemote); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "DNS updated"})
}

func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}
