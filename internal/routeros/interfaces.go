package routeros

import "net/url"

// Interface represents a RouterOS network interface
type Interface struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	MTU      string `json:"mtu"`
	MAC      string `json:"mac"`
	Running  bool   `json:"running"`
	Disabled bool   `json:"disabled"`
	Comment  string `json:"comment"`
	RxBytes  int64  `json:"rx_bytes"`
	TxBytes  int64  `json:"tx_bytes"`
	RxRate   int64  `json:"rx_rate"`
	TxRate   int64  `json:"tx_rate"`
}

// GetInterfaces returns all interfaces from the router
func (cm *ConnectionManager) GetInterfaces(routerID int) ([]Interface, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/interface")
	if err != nil {
		return nil, err
	}

	var interfaces []Interface
	for _, item := range results {
		iface := Interface{
			Name:     getString(item, "name"),
			Type:     getString(item, "type"),
			MTU:      getString(item, "mtu"),
			MAC:      getString(item, "mac-address"),
			Running:  getString(item, "running") == "true",
			Disabled: getString(item, "disabled") == "true",
			Comment:  getString(item, "comment"),
			RxBytes:  getInt64(item, "rx-byte"),
			TxBytes:  getInt64(item, "tx-byte"),
			RxRate:   getInt64(item, "rx-rate"),
			TxRate:   getInt64(item, "tx-rate"),
		}
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

// ToggleInterface enables or disables an interface
func (cm *ConnectionManager) ToggleInterface(routerID int, name string, disable bool) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	disabled := "false"
	if disable {
		disabled = "true"
	}

	data := url.Values{}
	data.Set("disabled", disabled)
	return restPut(conn, "/interface/"+name, data)
}
