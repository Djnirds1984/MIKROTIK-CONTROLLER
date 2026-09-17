package config

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	migrations := []string{
		// Routers table
		`CREATE TABLE IF NOT EXISTS routers (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL DEFAULT 80,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// PPPoE profiles (cached locally)
		`CREATE TABLE IF NOT EXISTS pppoe_profiles (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			local_address TEXT,
			remote_address TEXT,
			rate_limit TEXT,
			parent_profile TEXT,
			UNIQUE(router_id, name)
		)`,

		// PPPoE users (cached locally)
		`CREATE TABLE IF NOT EXISTS pppoe_users (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			password TEXT NOT NULL,
			profile TEXT NOT NULL DEFAULT 'default',
			service TEXT DEFAULT 'pppoe',
			comment TEXT,
			disabled BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(router_id, name)
		)`,

		// Traffic history
		`CREATE TABLE IF NOT EXISTS traffic_history (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
			interface_name TEXT NOT NULL,
			rx_bytes BIGINT DEFAULT 0,
			tx_bytes BIGINT DEFAULT 0,
			rx_rate BIGINT DEFAULT 0,
			tx_rate BIGINT DEFAULT 0,
			recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Index for traffic queries
		`CREATE INDEX IF NOT EXISTS idx_traffic_history_router_iface 
			ON traffic_history(router_id, interface_name, recorded_at)`,

		// PisoWiFi: Rate tiers
		`CREATE TABLE IF NOT EXISTS pisowifi_rates (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id),
			coin_value INTEGER NOT NULL,
			time_seconds INTEGER NOT NULL,
			label TEXT DEFAULT '',
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// PisoWiFi: Vouchers (separate from coin slot)
		`CREATE TABLE IF NOT EXISTS pisowifi_vouchers (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id),
			code TEXT UNIQUE NOT NULL,
			time_seconds INTEGER NOT NULL,
			status TEXT DEFAULT 'available',
			session_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP
		)`,

		// PisoWiFi: Active sessions
		`CREATE TABLE IF NOT EXISTS pisowifi_sessions (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id),
			username TEXT NOT NULL,
			mac_address TEXT,
			ip_address TEXT,
			total_seconds INTEGER NOT NULL,
			used_seconds INTEGER DEFAULT 0,
			remaining_seconds INTEGER NOT NULL,
			status TEXT DEFAULT 'active',
			paused_at TIMESTAMP,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP
		)`,

		// PisoWiFi: Coin log (tracks each coin inserted, bound to session)
		`CREATE TABLE IF NOT EXISTS pisowifi_coin_log (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id),
			session_id INTEGER NOT NULL REFERENCES pisowifi_sessions(id),
			device_id TEXT,
			coin_value INTEGER NOT NULL,
			time_added INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// PisoWiFi: NodeMCU device registry
		`CREATE TABLE IF NOT EXISTS pisowifi_devices (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id),
			device_id TEXT UNIQUE NOT NULL,
			name TEXT DEFAULT '',
			api_key TEXT,
			last_seen TIMESTAMP,
			status TEXT DEFAULT 'offline',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// PisoWiFi: Editable portal templates per router
		`CREATE TABLE IF NOT EXISTS pisowifi_portal_templates (
			id SERIAL PRIMARY KEY,
			router_id INTEGER NOT NULL REFERENCES routers(id) UNIQUE,
			html_content TEXT NOT NULL DEFAULT '',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w\nSQL: %s", err, m)
		}
	}

	return nil
}
