package routeros

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"mikrotik-controller/internal/models"
)

// RouterConn holds the HTTP client and base URL for a connected router
type RouterConn struct {
	Router     models.Router
	BaseURL    string
	HTTPClient *http.Client
}

type ConnectionManager struct {
	mu          sync.RWMutex
	connections map[int]*RouterConn
	routerInfo  map[int]*models.RouterStatus
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[int]*RouterConn),
		routerInfo:  make(map[int]*models.RouterStatus),
	}
}

func (cm *ConnectionManager) Connect(router models.Router) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Close existing connection if any
	if conn, ok := cm.connections[router.ID]; ok {
		conn.HTTPClient.CloseIdleConnections()
		delete(cm.connections, router.ID)
		delete(cm.routerInfo, router.ID)
	}

	baseURL := fmt.Sprintf("http://%s:%d/rest", router.Host, router.Port)

	// Test connection by fetching system identity
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", baseURL+"/system/identity", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(router.Username, router.Password)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	conn := &RouterConn{
		Router:     router,
		BaseURL:    baseURL,
		HTTPClient: client,
	}

	cm.connections[router.ID] = conn
	cm.routerInfo[router.ID] = &models.RouterStatus{
		Router:    router,
		Connected: true,
	}

	return nil
}

func (cm *ConnectionManager) Disconnect(routerID int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, ok := cm.connections[routerID]; ok {
		conn.HTTPClient.CloseIdleConnections()
		delete(cm.connections, routerID)
		delete(cm.routerInfo, routerID)
	}
	// Clean up cached traffic history
	trafficMu.Lock()
	delete(trafficHistory, routerID)
	trafficMu.Unlock()
}

func (cm *ConnectionManager) GetConn(routerID int) (*RouterConn, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, ok := cm.connections[routerID]
	if !ok {
		return nil, fmt.Errorf("router %d not connected", routerID)
	}
	return conn, nil
}

func (cm *ConnectionManager) IsConnected(routerID int) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, ok := cm.connections[routerID]
	return ok
}

func (cm *ConnectionManager) GetStatus(routerID int) *models.RouterStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.routerInfo[routerID]
}

func (cm *ConnectionManager) UpdateStatus(routerID int, status *models.RouterStatus) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.routerInfo[routerID] = status
}

func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for id, conn := range cm.connections {
		conn.HTTPClient.CloseIdleConnections()
		delete(cm.connections, id)
	}
}

// restGet performs a GET request to the RouterOS REST API
func restGet(conn *RouterConn, path string) ([]map[string]interface{}, error) {
	req, err := http.NewRequest("GET", conn.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(conn.Router.Username, conn.Router.Password)

	resp, err := conn.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// Some endpoints return a single object, not an array
		var single map[string]interface{}
		if err2 := json.Unmarshal(body, &single); err2 == nil {
			return []map[string]interface{}{single}, nil
		}
		return nil, fmt.Errorf("parse response: %w (body: %s)", err, string(body))
	}
	return result, nil
}

// restPut performs a PUT request to the RouterOS REST API
func restPut(conn *RouterConn, path string, data url.Values) error {
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest("PUT", conn.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(conn.Router.Username, conn.Router.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := conn.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// restPost performs a POST request to the RouterOS REST API
func restPost(conn *RouterConn, path string, data url.Values) error {
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest("POST", conn.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(conn.Router.Username, conn.Router.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := conn.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// restDelete performs a DELETE request to the RouterOS REST API
func restDelete(conn *RouterConn, path string) error {
	req, err := http.NewRequest("DELETE", conn.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(conn.Router.Username, conn.Router.Password)

	resp, err := conn.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// getString safely extracts a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case float64:
			return fmt.Sprintf("%.0f", val)
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

// getFloat64 safely extracts a float64 from a map
func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case string:
			f, _ := strconv.ParseFloat(val, 64)
			return f
		}
	}
	return 0
}

// getInt64 safely extracts an int64 from a map
func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int64(val)
		case string:
			n, _ := strconv.ParseInt(val, 10, 64)
			return n
		}
	}
	return 0
}

// StartTrafficCollector periodically collects traffic data from connected routers
func (cm *ConnectionManager) StartTrafficCollector(db *sql.DB) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cm.mu.RLock()
		routerIDs := make([]int, 0, len(cm.connections))
		for id := range cm.connections {
			routerIDs = append(routerIDs, id)
		}
		cm.mu.RUnlock()

		for _, routerID := range routerIDs {
			cm.collectTraffic(db, routerID)
		}
	}
}

func (cm *ConnectionManager) collectTraffic(db *sql.DB, routerID int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Traffic collector panic recovered for router %d: %v", routerID, r)
		}
	}()

	conn, err := cm.GetConn(routerID)
	if err != nil {
		return
	}

	results, err := restGet(conn, "/interface")
	if err != nil {
		log.Printf("Traffic collection error for router %d: %v", routerID, err)
		return
	}

	now := time.Now()
	for _, item := range results {
		ifaceName := getString(item, "name")
		rxBytes := getInt64(item, "rx-byte")
		txBytes := getInt64(item, "tx-byte")

		var prevRx, prevTx int64
		db.QueryRow(
			"SELECT rx_bytes, tx_bytes FROM traffic_history WHERE router_id = $1 AND interface_name = $2 ORDER BY recorded_at DESC LIMIT 1",
			routerID, ifaceName,
		).Scan(&prevRx, &prevTx)

		rxRate := int64(0)
		txRate := int64(0)
		if prevRx > 0 {
			rxRate = (rxBytes - prevRx) / 30
			txRate = (txBytes - prevTx) / 30
			if rxRate < 0 {
				rxRate = 0
			}
			if txRate < 0 {
				txRate = 0
			}
		}

		db.Exec(
			"INSERT INTO traffic_history (router_id, interface_name, rx_bytes, tx_bytes, rx_rate, tx_rate, recorded_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			routerID, ifaceName, rxBytes, txBytes, rxRate, txRate, now,
		)
	}
}
