package models

import "time"

type Router struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RouterStatus struct {
	Router        Router
	Connected     bool
	BoardName     string
	FirmwareVersion string
	Uptime        int64
	CPULoad       int
	FreeMemory    int64
	TotalMemory   int64
	Temperature   float64
	Identity      string
}
