package routeros

import (
	"fmt"
	"net/url"
)

// HotspotUser represents a MikroTik hotspot user
type HotspotUser struct {
	Name        string `json:"name"`
	Password    string `json:"password"`
	Profile     string `json:"profile"`
	Comment     string `json:"comment"`
	Disabled    bool   `json:"disabled"`
	Uptime      string `json:"uptime"`
	BytesIn     int64  `json:"bytes_in"`
	BytesOut    int64  `json:"bytes_out"`
	LimitUptime string `json:"limit_uptime"`
	LimitBytes  string `json:"limit_bytes"`
}

// HotspotProfile represents a hotspot user profile
type HotspotProfile struct {
	Name              string `json:"name"`
	RateLimit         string `json:"rate_limit"`
	SessionTimeout    string `json:"session_timeout"`
	IdleTimeout       string `json:"idle_timeout"`
	SharedUsers       string `json:"shared_users"`
	StatusAutorefresh string `json:"status_autorefresh"`
	LoginBy           string `json:"login_by"`
}

// HotspotActive represents an active hotspot session
type HotspotActive struct {
	Username string `json:"username"`
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	LoginBy  string `json:"login_by"`
	Uptime   string `json:"uptime"`
	BytesIn  int64  `json:"bytes_in"`
	BytesOut int64  `json:"bytes_out"`
	IdleTime string `json:"idle_time"`
	HostIP   string `json:"host_ip"`
	Server   string `json:"server"`
}

// HotspotServer represents a hotspot server config
type HotspotServer struct {
	Name        string `json:"name"`
	Interface   string `json:"interface"`
	AddressPool string `json:"address_pool"`
	Disabled    bool   `json:"disabled"`
}

// FormatUptime converts seconds to MikroTik uptime format (e.g. "1h30m", "30m", "30s")
func FormatUptime(seconds int) string {
	if seconds <= 0 {
		return "0s"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	result := ""
	if h > 0 {
		result += fmt.Sprintf("%dh", h)
	}
	if m > 0 {
		result += fmt.Sprintf("%dm", m)
	}
	if s > 0 && h == 0 {
		result += fmt.Sprintf("%ds", s)
	}
	if result == "" {
		return "0s"
	}
	return result
}

// GetHotspotUsers retrieves all hotspot users
func (cm *ConnectionManager) GetHotspotUsers(routerID int) ([]HotspotUser, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ip/hotspot/user")
	if err != nil {
		return nil, err
	}

	var users []HotspotUser
	for _, item := range results {
		user := HotspotUser{
			Name:        getString(item, "name"),
			Password:    getString(item, "password"),
			Profile:     getString(item, "profile"),
			Comment:     getString(item, "comment"),
			Disabled:    getString(item, "disabled") == "true",
			Uptime:      getString(item, "uptime"),
			BytesIn:     getInt64(item, "bytes-in"),
			BytesOut:    getInt64(item, "bytes-out"),
			LimitUptime: getString(item, "limit-uptime"),
			LimitBytes:  getString(item, "limit-bytes"),
		}
		users = append(users, user)
	}

	return users, nil
}

// AddHotspotUser adds a new hotspot user
func (cm *ConnectionManager) AddHotspotUser(routerID int, user HotspotUser) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("name", user.Name)
	data.Set("password", user.Password)
	data.Set("profile", user.Profile)
	if user.Comment != "" {
		data.Set("comment", user.Comment)
	}
	if user.Disabled {
		data.Set("disabled", "true")
	}
	if user.LimitUptime != "" {
		data.Set("limit-uptime", user.LimitUptime)
	}
	if user.LimitBytes != "" {
		data.Set("limit-bytes", user.LimitBytes)
	}
	return restPost(conn, "/ip/hotspot/user", data)
}

// UpdateHotspotUser updates an existing hotspot user
func (cm *ConnectionManager) UpdateHotspotUser(routerID int, oldName string, user HotspotUser) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("name", user.Name)
	data.Set("password", user.Password)
	data.Set("profile", user.Profile)
	if user.Comment != "" {
		data.Set("comment", user.Comment)
	}
	if user.Disabled {
		data.Set("disabled", "true")
	} else {
		data.Set("disabled", "false")
	}
	if user.LimitUptime != "" {
		data.Set("limit-uptime", user.LimitUptime)
	}
	if user.LimitBytes != "" {
		data.Set("limit-bytes", user.LimitBytes)
	}
	return restPut(conn, "/ip/hotspot/user/"+oldName, data)
}

// DeleteHotspotUser removes a hotspot user
func (cm *ConnectionManager) DeleteHotspotUser(routerID int, name string) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	return restDelete(conn, "/ip/hotspot/user/"+name)
}

// GetHotspotProfiles retrieves all hotspot user profiles
func (cm *ConnectionManager) GetHotspotProfiles(routerID int) ([]HotspotProfile, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ip/hotspot/user/profile")
	if err != nil {
		return nil, err
	}

	var profiles []HotspotProfile
	for _, item := range results {
		profile := HotspotProfile{
			Name:           getString(item, "name"),
			RateLimit:      getString(item, "rate-limit"),
			SessionTimeout: getString(item, "session-timeout"),
			IdleTimeout:    getString(item, "idle-timeout"),
			SharedUsers:    getString(item, "shared-users"),
			LoginBy:        getString(item, "login-by"),
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// AddHotspotProfile creates a new hotspot user profile
func (cm *ConnectionManager) AddHotspotProfile(routerID int, profile HotspotProfile) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("name", profile.Name)
	if profile.RateLimit != "" {
		data.Set("rate-limit", profile.RateLimit)
	}
	if profile.SessionTimeout != "" {
		data.Set("session-timeout", profile.SessionTimeout)
	}
	if profile.IdleTimeout != "" {
		data.Set("idle-timeout", profile.IdleTimeout)
	}
	if profile.SharedUsers != "" {
		data.Set("shared-users", profile.SharedUsers)
	}
	if profile.LoginBy != "" {
		data.Set("login-by", profile.LoginBy)
	}
	return restPost(conn, "/ip/hotspot/user/profile", data)
}

// GetHotspotActive retrieves active hotspot sessions
func (cm *ConnectionManager) GetHotspotActive(routerID int) ([]HotspotActive, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ip/hotspot/active")
	if err != nil {
		return nil, err
	}

	var sessions []HotspotActive
	for _, item := range results {
		session := HotspotActive{
			Username: getString(item, "user"),
			IP:       getString(item, "address"),
			MAC:      getString(item, "mac-address"),
			LoginBy:  getString(item, "login-by"),
			Uptime:   getString(item, "uptime"),
			BytesIn:  getInt64(item, "bytes-in"),
			BytesOut: getInt64(item, "bytes-out"),
			IdleTime: getString(item, "idle-time"),
			HostIP:   getString(item, "host-ip"),
			Server:   getString(item, "server"),
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetHotspotServers retrieves hotspot server configurations
func (cm *ConnectionManager) GetHotspotServers(routerID int) ([]HotspotServer, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ip/hotspot")
	if err != nil {
		return nil, err
	}

	var servers []HotspotServer
	for _, item := range results {
		server := HotspotServer{
			Name:        getString(item, "name"),
			Interface:   getString(item, "interface"),
			AddressPool: getString(item, "address-pool"),
			Disabled:    getString(item, "disabled") == "true",
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// SetupHotspotRedirect configures MikroTik to redirect all HTTP traffic to the SBC portal
// This adds:
// 1. A walled garden entry allowing the SBC IP
// 2. A NAT dst-nat rule redirecting HTTP to the SBC
func (cm *ConnectionManager) SetupHotspotRedirect(routerID int, sbcIP string, sbcPort int) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	// 1. Add walled garden entry for SBC IP (allows unauthenticated users to reach SBC)
	wgData := url.Values{}
	wgData.Set("action", "accept")
	wgData.Set("dst-host", sbcIP)
	wgData.Set("dst-port", fmt.Sprintf("%d", sbcPort))
	wgData.Set("comment", "pisowifi-sbc-portal")
	if err := restPost(conn, "/ip/hotspot/walled-garden", wgData); err != nil {
		return fmt.Errorf("walled garden: %w", err)
	}

	// 2. Add NAT dst-nat rule to redirect all HTTP (port 80) to SBC portal
	natData := url.Values{}
	natData.Set("chain", "dstnat")
	natData.Set("action", "dst-nat")
	natData.Set("protocol", "tcp")
	natData.Set("dst-port", "80")
	natData.Set("to-addresses", sbcIP)
	natData.Set("to-ports", fmt.Sprintf("%d", sbcPort))
	natData.Set("comment", "pisowifi-redirect")
	if err := restPost(conn, "/ip/firewall/nat", natData); err != nil {
		return fmt.Errorf("nat rule: %w", err)
	}

	return nil
}

// HotspotSetupStatus holds the setup check results for a MikroTik router
type HotspotSetupStatus struct {
	HotspotRunning   bool   `json:"hotspot_running"`
	WalledGarden     bool   `json:"walled_garden"`
	NatRedirect      bool   `json:"nat_redirect"`
	FullyConfigured  bool   `json:"fully_configured"`
	WalledGardenHost string `json:"walled_garden_host,omitempty"`
}

// CheckHotspotSetup checks if the MikroTik router has the required PisoWiFi configuration
func (cm *ConnectionManager) CheckHotspotSetup(routerID int) (*HotspotSetupStatus, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	status := &HotspotSetupStatus{}

	// 1. Check if hotspot server is running
	servers, err := restGet(conn, "/ip/hotspot")
	if err == nil && len(servers) > 0 {
		for _, s := range servers {
			if getString(s, "disabled") != "true" {
				status.HotspotRunning = true
				break
			}
		}
	}

	// 2. Check walled garden for pisowifi entry
	wgResults, err := restGet(conn, "/ip/hotspot/walled-garden")
	if err == nil {
		for _, item := range wgResults {
			if getString(item, "comment") == "pisowifi-sbc-portal" {
				status.WalledGarden = true
				status.WalledGardenHost = getString(item, "dst-host")
				break
			}
		}
	}

	// 3. Check NAT dst-nat rule for pisowifi redirect
	natResults, err := restGet(conn, "/ip/firewall/nat")
	if err == nil {
		for _, item := range natResults {
			if getString(item, "comment") == "pisowifi-redirect" {
				status.NatRedirect = true
				break
			}
		}
	}

	status.FullyConfigured = status.HotspotRunning && status.WalledGarden && status.NatRedirect
	return status, nil
}

// RemoveHotspotRedirect removes the pisowifi redirect rules from MikroTik
func (cm *ConnectionManager) RemoveHotspotRedirect(routerID int) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	// Remove walled garden entries with pisowifi comment
	wgResults, err := restGet(conn, "/ip/hotspot/walled-garden")
	if err == nil {
		for _, item := range wgResults {
			if getString(item, "comment") == "pisowifi-sbc-portal" {
				id := getString(item, ".id")
				restDelete(conn, "/ip/hotspot/walled-garden/"+id)
			}
		}
	}

	// Remove NAT rules with pisowifi comment
	natResults, err := restGet(conn, "/ip/firewall/nat")
	if err == nil {
		for _, item := range natResults {
			if getString(item, "comment") == "pisowifi-redirect" {
				id := getString(item, ".id")
				restDelete(conn, "/ip/firewall/nat/"+id)
			}
		}
	}

	return nil
}
