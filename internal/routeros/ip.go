package routeros

import "net/url"

// IPAddress represents an IP address assigned to an interface
type IPAddress struct {
	Address   string `json:"address"`
	Interface string `json:"interface"`
	Network   string `json:"network"`
	Disabled  bool   `json:"disabled"`
	Comment   string `json:"comment"`
}

// DHCPConfig represents DHCP server configuration
type DHCPConfig struct {
	Name        string `json:"name"`
	Interface   string `json:"interface"`
	AddressPool string `json:"address_pool"`
	LeaseTime   string `json:"lease_time"`
	Disabled    bool   `json:"disabled"`
}

// DNSConfig represents DNS settings
type DNSConfig struct {
	Servers        []string `json:"servers"`
	DynamicServers []string `json:"dynamic_servers"`
	AllowRemote    bool     `json:"allow_remote"`
}

// GetIPAddresses returns all IP addresses from the router
func (cm *ConnectionManager) GetIPAddresses(routerID int) ([]IPAddress, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ip/address")
	if err != nil {
		return nil, err
	}

	var addresses []IPAddress
	for _, item := range results {
		addr := IPAddress{
			Address:   getString(item, "address"),
			Interface: getString(item, "interface"),
			Network:   getString(item, "network"),
			Disabled:  getString(item, "disabled") == "true",
			Comment:   getString(item, "comment"),
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// AddIPAddress adds an IP address to an interface
func (cm *ConnectionManager) AddIPAddress(routerID int, address, iface string) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("address", address)
	data.Set("interface", iface)
	return restPost(conn, "/ip/address", data)
}

// DeleteIPAddress removes an IP address
func (cm *ConnectionManager) DeleteIPAddress(routerID int, address string) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	return restDelete(conn, "/ip/address/"+address)
}

// GetDHCPConfig returns DHCP server configurations
func (cm *ConnectionManager) GetDHCPConfig(routerID int) ([]DHCPConfig, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ip/dhcp-server")
	if err != nil {
		return nil, err
	}

	var configs []DHCPConfig
	for _, item := range results {
		cfg := DHCPConfig{
			Name:        getString(item, "name"),
			Interface:   getString(item, "interface"),
			AddressPool: getString(item, "address-pool"),
			LeaseTime:   getString(item, "lease-time"),
			Disabled:    getString(item, "disabled") == "true",
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}

// GetDNSConfig returns DNS configuration
func (cm *ConnectionManager) GetDNSConfig(routerID int) (*DNSConfig, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ip/dns")
	if err != nil {
		return nil, err
	}

	cfg := &DNSConfig{}
	if len(results) > 0 {
		r := results[0]
		if servers := getString(r, "servers"); servers != "" {
			cfg.Servers = splitAndTrim(servers, ",")
		}
		cfg.AllowRemote = getString(r, "allow-remote-requests") == "true"
	}

	return cfg, nil
}

// UpdateDNS updates DNS settings
func (cm *ConnectionManager) UpdateDNS(routerID int, servers string, allowRemote bool) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("servers", servers)
	if allowRemote {
		data.Set("allow-remote-requests", "true")
	} else {
		data.Set("allow-remote-requests", "false")
	}
	return restPut(conn, "/ip/dns", data)
}

func splitAndTrim(s string, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			part := s[start:i]
			if len(part) > 0 {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
