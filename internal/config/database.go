package config

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL DEFAULT 8728,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// PPPoE profiles (cached locally)
		`CREATE TABLE IF NOT EXISTS pppoe_profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			router_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			local_address TEXT,
			remote_address TEXT,
			rate_limit TEXT,
			parent_profile TEXT,
			FOREIGN KEY (router_id) REFERENCES routers(id) ON DELETE CASCADE,
			UNIQUE(router_id, name)
		)`,

		// PPPoE users (cached locally)
		`CREATE TABLE IF NOT EXISTS pppoe_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			router_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			password TEXT NOT NULL,
			profile TEXT NOT NULL DEFAULT 'default',
			service TEXT DEFAULT 'pppoe',
			comment TEXT,
			disabled INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (router_id) REFERENCES routers(id) ON DELETE CASCADE,
			UNIQUE(router_id, name)
		)`,

		// Traffic history
		`CREATE TABLE IF NOT EXISTS traffic_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			router_id INTEGER NOT NULL,
			interface_name TEXT NOT NULL,
			rx_bytes INTEGER DEFAULT 0,
			tx_bytes INTEGER DEFAULT 0,
			rx_rate INTEGER DEFAULT 0,
			tx_rate INTEGER DEFAULT 0,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (router_id) REFERENCES routers(id) ON DELETE CASCADE
		)`,

		// Index for traffic queries
		`CREATE INDEX IF NOT EXISTS idx_traffic_history_router_iface 
			ON traffic_history(router_id, interface_name, recorded_at)`,

		// Schema version tracking
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w\nSQL: %s", err, m)
		}
	}

	// Set schema version
	db.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (1)")

	return nil
}
