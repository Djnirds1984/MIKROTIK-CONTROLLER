package routeros

import "net/url"

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
