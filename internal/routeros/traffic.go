package routeros

import (
	"log"
	"sync"
	"time"
)

// TrafficData represents real-time traffic data for an interface
type TrafficData struct {
	Interface string `json:"interface"`
	RxBytes   int64  `json:"rx_bytes"`
	TxBytes   int64  `json:"tx_bytes"`
	RxRate    int64  `json:"rx_rate"`
	TxRate    int64  `json:"tx_rate"`
}

// ClientInfo represents a connected client
type ClientInfo struct {
	IPAddress string `json:"ip_address"`
	MAC       string `json:"mac"`
	Interface string `json:"interface"`
	Host      string `json:"host"`
}

// trafficSnapshot stores previous byte counts for rate calculation
type trafficSnapshot struct {
	rxBytes int64
	txBytes int64
	time    time.Time
}

var (
	trafficMu      sync.Mutex
	trafficHistory = make(map[int]map[string]*trafficSnapshot) // routerID -> interface -> snapshot
)

// GetTraffic returns real-time traffic data for all interfaces with calculated rates
func (cm *ConnectionManager) GetTraffic(routerID int) ([]TrafficData, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/interface")
	if err != nil {
		return nil, err
	}

	trafficMu.Lock()
	if trafficHistory[routerID] == nil {
		trafficHistory[routerID] = make(map[string]*trafficSnapshot)
	}
	prevMap := trafficHistory[routerID]
	now := time.Now()

	var traffic []TrafficData
	newSnap := make(map[string]*trafficSnapshot)

	for _, item := range results {
		name := getString(item, "name")
		if name == "" {
			continue
		}

		rxBytes := getInt64(item, "rx-byte")
		txBytes := getInt64(item, "tx-byte")

		// Also try alternative field names
		if rxBytes == 0 {
			rxBytes = getInt64(item, "rx-bytes")
		}
		if txBytes == 0 {
			txBytes = getInt64(item, "tx-bytes")
		}

		var rxRate, txRate int64

		// Calculate rate from previous snapshot
		if prev, ok := prevMap[name]; ok {
			elapsed := now.Sub(prev.time).Seconds()
			if elapsed > 0 {
				rxDiff := rxBytes - prev.rxBytes
				txDiff := txBytes - prev.txBytes
				if rxDiff < 0 {
					rxDiff = 0
				}
				if txDiff < 0 {
					txDiff = 0
				}
				rxRate = int64(float64(rxDiff) / elapsed)
				txRate = int64(float64(txDiff) / elapsed)
			}
		}

		newSnap[name] = &trafficSnapshot{
			rxBytes: rxBytes,
			txBytes: txBytes,
			time:    now,
		}

		traffic = append(traffic, TrafficData{
			Interface: name,
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			RxRate:    rxRate,
			TxRate:    txRate,
		})
	}

	trafficHistory[routerID] = newSnap
	trafficMu.Unlock()

	return traffic, nil
}

// GetClients returns connected clients from ARP table
func (cm *ConnectionManager) GetClients(routerID int) ([]ClientInfo, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ip/arp")
	if err != nil {
		return nil, err
	}

	var clients []ClientInfo
	for _, item := range results {
		c := ClientInfo{
			IPAddress: getString(item, "address"),
			MAC:       getString(item, "mac-address"),
			Interface: getString(item, "interface"),
			Host:      getString(item, "host-name"),
		}
		clients = append(clients, c)
	}

	return clients, nil
}

// CleanupTrafficHistory removes cached data for a disconnected router
func (cm *ConnectionManager) CleanupTrafficHistory(routerID int) {
	trafficMu.Lock()
	delete(trafficHistory, routerID)
	trafficMu.Unlock()
	log.Printf("Cleaned up traffic history for router %d", routerID)
}
