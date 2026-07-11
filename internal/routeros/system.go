package routeros

import (
	"strconv"

	"mikrotik-controller/internal/models"
)

// GetSystemInfo retrieves system information from a connected router
func (cm *ConnectionManager) GetSystemInfo(routerID int) (*models.RouterStatus, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	status := cm.GetStatus(routerID)
	if status == nil {
		status = &models.RouterStatus{Connected: true}
	}

	// Get system identity
	results, err := restGet(conn, "/system/identity")
	if err == nil && len(results) > 0 {
		status.Identity = getString(results[0], "name")
	}

	// Get resource info
	results, err = restGet(conn, "/system/resource")
	if err == nil && len(results) > 0 {
		r := results[0]
		status.BoardName = getString(r, "board-name")
		status.FirmwareVersion = getString(r, "version")
		if identity := getString(r, "identity"); identity != "" {
			status.Identity = identity
		}

		if uptimeStr := getString(r, "uptime"); uptimeStr != "" {
			if uptime, err := parseUptime(uptimeStr); err == nil {
				status.Uptime = uptime
			}
		}
		if cpu := getString(r, "cpu-load"); cpu != "" {
			if cpuLoad, err := strconv.Atoi(cpu); err == nil {
				status.CPULoad = cpuLoad
			}
		}
		status.FreeMemory = getInt64(r, "free-memory")
		status.TotalMemory = getInt64(r, "total-memory")
	}

	// Get health info (temperature)
	results, err = restGet(conn, "/system/health")
	if err == nil && len(results) > 0 {
		status.Temperature = getFloat64(results[0], "temperature")
	}

	status.Connected = true
	cm.UpdateStatus(routerID, status)
	return status, nil
}

// parseUptime parses RouterOS uptime format (e.g., "1d2h3m4s" or "1w2d3h")
func parseUptime(s string) (int64, error) {
	var total int64
	var current int64

	for _, c := range s {
		switch c {
		case 'w':
			total += current * 7 * 86400
			current = 0
		case 'd':
			total += current * 86400
			current = 0
		case 'h':
			total += current * 3600
			current = 0
		case 'm':
			total += current * 60
			current = 0
		case 's':
			total += current
			current = 0
		default:
			digit, err := strconv.Atoi(string(c))
			if err != nil {
				return 0, err
			}
			current = current*10 + int64(digit)
		}
	}
	total += current
	return total, nil
}
