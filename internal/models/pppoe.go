package models

type PPPoEUser struct {
	ID       int    `json:"id"`
	RouterID int    `json:"router_id"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Profile  string `json:"profile"`
	Service  string `json:"service"`
	Comment  string `json:"comment"`
	Disabled bool   `json:"disabled"`
}

type PPPoEProfile struct {
	ID            int    `json:"id"`
	RouterID      int    `json:"router_id"`
	Name          string `json:"name"`
	LocalAddress  string `json:"local_address"`
	RemoteAddress string `json:"remote_address"`
	RateLimit     string `json:"rate_limit"`
	ParentProfile string `json:"parent_profile"`
}

type PPPoESession struct {
	Username   string `json:"username"`
	Interface  string `json:"interface"`
	Service    string `json:"service"`
	State      string `json:"state"`
	Uptime     int64  `json:"uptime"`
	CallerID   string `json:"caller_id"`
	IPAddress  string `json:"ip_address"`
}
