package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"mikrotik-controller/internal/config"
	"mikrotik-controller/internal/handlers"
	"mikrotik-controller/internal/pisowifi"
	"mikrotik-controller/internal/routeros"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := config.InitDB(cfg.DSN())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize RouterOS connection manager
	connMgr := routeros.NewConnectionManager()
	defer connMgr.CloseAll()

	// Parse templates
	templates, err := parseTemplates()
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// Initialize handlers
	h := handlers.New(templates, db, connMgr)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Serve static files from embedded FS
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Page routes
	mux.HandleFunc("GET /{$}", h.Dashboard)
	mux.HandleFunc("GET /routers", h.RoutersPage)
	mux.HandleFunc("GET /interfaces", h.InterfacesPage)
	mux.HandleFunc("GET /pppoe", h.PPPoEPage)
	mux.HandleFunc("GET /hotspot", h.HotspotPage)
	mux.HandleFunc("GET /traffic", h.TrafficPage)
	mux.HandleFunc("GET /pisowifi", h.PisoWifiPage)

	// Portal routes (public, standalone)
	mux.HandleFunc("GET /portal/{router_id}", h.PortalPage)
	mux.HandleFunc("POST /portal/{router_id}/redeem", h.PortalRedeem)
	mux.HandleFunc("GET /portal/{router_id}/session", h.PortalSession)
	mux.HandleFunc("POST /portal/{router_id}/pause", h.PortalPause)
	mux.HandleFunc("POST /portal/{router_id}/resume", h.PortalResume)
	mux.HandleFunc("POST /portal/{router_id}/logout", h.PortalLogout)

	// Coin event API (NodeMCU)
	mux.HandleFunc("POST /api/coin-event", h.HandleCoinEvent)

	// API routes (HTMX partials)
	mux.HandleFunc("POST /api/routers", h.AddRouter)
	mux.HandleFunc("PUT /api/routers/{id}", h.UpdateRouter)
	mux.HandleFunc("DELETE /api/routers/{id}", h.DeleteRouter)
	mux.HandleFunc("POST /api/routers/{id}/connect", h.ConnectRouter)
	mux.HandleFunc("POST /api/routers/{id}/disconnect", h.DisconnectRouter)

	// Interface API
	mux.HandleFunc("GET /api/routers/{id}/interfaces", h.GetInterfaces)
	mux.HandleFunc("PUT /api/routers/{id}/interfaces/{name}", h.ToggleInterface)
	mux.HandleFunc("GET /api/routers/{id}/ip-addresses", h.GetIPAddresses)
	mux.HandleFunc("POST /api/routers/{id}/ip-addresses", h.AddIPAddress)
	mux.HandleFunc("DELETE /api/routers/{id}/ip-addresses/{address}", h.DeleteIPAddress)
	mux.HandleFunc("GET /api/routers/{id}/dhcp", h.GetDHCPConfig)
	mux.HandleFunc("GET /api/routers/{id}/dns", h.GetDNSConfig)
	mux.HandleFunc("PUT /api/routers/{id}/dns", h.UpdateDNS)

	// PPPoE API
	mux.HandleFunc("GET /api/routers/{id}/pppoe/users", h.GetPPPoEUsers)
	mux.HandleFunc("POST /api/routers/{id}/pppoe/users", h.AddPPPoEUser)
	mux.HandleFunc("PUT /api/routers/{id}/pppoe/users/{name}", h.UpdatePPPoEUser)
	mux.HandleFunc("DELETE /api/routers/{id}/pppoe/users/{name}", h.DeletePPPoEUser)
	mux.HandleFunc("POST /api/routers/{id}/pppoe/import", h.ImportPPPoEUsers)
	mux.HandleFunc("GET /api/routers/{id}/pppoe/profiles", h.GetPPPoEProfiles)
	mux.HandleFunc("POST /api/routers/{id}/pppoe/profiles", h.AddPPPoEProfile)
	mux.HandleFunc("GET /api/routers/{id}/pppoe/sessions", h.GetPPPoESessions)

	// Traffic API
	mux.HandleFunc("GET /api/routers/{id}/traffic", h.GetTraffic)
	mux.HandleFunc("GET /api/routers/{id}/traffic/clients", h.GetTrafficClients)
	mux.HandleFunc("GET /api/routers/{id}/traffic/history", h.GetTrafficHistory)

	// Hotspot API
	mux.HandleFunc("GET /api/routers/{id}/hotspot/users", h.GetHotspotUsers)
	mux.HandleFunc("POST /api/routers/{id}/hotspot/users", h.AddHotspotUser)
	mux.HandleFunc("PUT /api/routers/{id}/hotspot/users/{name}", h.UpdateHotspotUser)
	mux.HandleFunc("DELETE /api/routers/{id}/hotspot/users/{name}", h.DeleteHotspotUser)
	mux.HandleFunc("POST /api/routers/{id}/hotspot/import", h.ImportHotspotUsers)
	mux.HandleFunc("GET /api/routers/{id}/hotspot/profiles", h.GetHotspotProfiles)
	mux.HandleFunc("POST /api/routers/{id}/hotspot/profiles", h.AddHotspotProfile)
	mux.HandleFunc("GET /api/routers/{id}/hotspot/active", h.GetHotspotActive)
	mux.HandleFunc("GET /api/routers/{id}/hotspot/servers", h.GetHotspotServers)

	// PisoWiFi Admin API
	mux.HandleFunc("GET /api/routers/{id}/pisowifi/rates", h.GetPisoRates)
	mux.HandleFunc("POST /api/routers/{id}/pisowifi/rates", h.AddPisoRate)
	mux.HandleFunc("PUT /api/routers/{id}/pisowifi/rates/{rateId}", h.UpdatePisoRate)
	mux.HandleFunc("DELETE /api/routers/{id}/pisowifi/rates/{rateId}", h.DeletePisoRate)

	mux.HandleFunc("GET /api/routers/{id}/pisowifi/sessions", h.GetPisoSessions)
	mux.HandleFunc("POST /api/routers/{id}/pisowifi/sessions/{sid}/pause", h.PausePisoSession)
	mux.HandleFunc("POST /api/routers/{id}/pisowifi/sessions/{sid}/resume", h.ResumePisoSession)
	mux.HandleFunc("POST /api/routers/{id}/pisowifi/sessions/{sid}/disconnect", h.DisconnectPisoSession)
	mux.HandleFunc("POST /api/routers/{id}/pisowifi/sessions/{sid}/extend", h.ExtendPisoSession)

	mux.HandleFunc("GET /api/routers/{id}/pisowifi/vouchers", h.GetPisoVouchers)
	mux.HandleFunc("POST /api/routers/{id}/pisowifi/vouchers/generate", h.GeneratePisoVoucher)
	mux.HandleFunc("POST /api/routers/{id}/pisowifi/vouchers/batch", h.BatchGeneratePisoVouchers)
	mux.HandleFunc("GET /api/routers/{id}/pisowifi/vouchers/export", h.ExportPisoVouchers)

	mux.HandleFunc("GET /api/routers/{id}/pisowifi/earnings", h.GetPisoEarnings)

	mux.HandleFunc("GET /api/routers/{id}/pisowifi/devices", h.GetPisoDevices)
	mux.HandleFunc("POST /api/routers/{id}/pisowifi/devices", h.RegisterPisoDevice)
	mux.HandleFunc("DELETE /api/routers/{id}/pisowifi/devices/{devId}", h.DeletePisoDevice)

	// Start PisoWiFi session manager
	sessionMgr := pisowifi.NewSessionManager(db)
	sessionMgr.OnExpire = func(s pisowifi.Session) {
		h.SyncDisableHotspotUser(s.RouterID, s.Username)
	}
	sessionMgr.Start()
	defer sessionMgr.Stop()

	// Start traffic collector
	go connMgr.StartTrafficCollector(db)

	// Setup server with panic recovery middleware
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: recoveryMiddleware(mux),
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		server.Close()
	}()

	log.Printf("MikroTik Controller starting on http://0.0.0.0:%d", cfg.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func parseTemplates() (map[string]*template.Template, error) {
	funcMap := template.FuncMap{
		"formatBytes":  formatBytes,
		"formatUptime": formatUptime,
		"int": func(v interface{}) int {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case string:
				n, _ := strconv.Atoi(val)
				return n
			default:
				return 0
			}
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	pages := []string{"dashboard.html", "routers.html", "interfaces.html", "pppoe.html", "hotspot.html", "traffic.html", "pisowifi.html"}
	templates := make(map[string]*template.Template)

	for _, page := range pages {
		// Each page gets its own template instance with layout + page content
		tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/layout.html", "templates/"+page)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		templates[page] = tmpl
	}

	// Portal template is standalone (no layout)
	portalTmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/portal.html")
	if err != nil {
		return nil, fmt.Errorf("parse portal.html: %w", err)
	}
	templates["portal.html"] = portalTmpl

	return templates, nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// recoveryMiddleware catches any panics in HTTP handlers and returns a 500 error
// instead of crashing the server.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("HTTP handler panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
