package pisowifi

import (
	"database/sql"
	"time"
)

// PortalTemplate represents a customizable portal HTML template for a router
type PortalTemplate struct {
	ID        int       `json:"id"`
	RouterID  int       `json:"router_id"`
	HTML      string    `json:"html"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PortalTemplateStore handles CRUD for portal templates
type PortalTemplateStore struct {
	db *sql.DB
}

func NewPortalTemplateStore(db *sql.DB) *PortalTemplateStore {
	return &PortalTemplateStore{db: db}
}

// GetTemplate returns the custom portal template for a router, or empty if none exists
func (s *PortalTemplateStore) GetTemplate(routerID int) (*PortalTemplate, error) {
	var t PortalTemplate
	err := s.db.QueryRow(
		"SELECT id, router_id, html_content, updated_at FROM pisowifi_portal_templates WHERE router_id = $1",
		routerID,
	).Scan(&t.ID, &t.RouterID, &t.HTML, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// SaveTemplate creates or updates the portal template for a router
func (s *PortalTemplateStore) SaveTemplate(routerID int, html string) (*PortalTemplate, error) {
	var t PortalTemplate
	err := s.db.QueryRow(
		`INSERT INTO pisowifi_portal_templates (router_id, html_content, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (router_id) DO UPDATE SET html_content = $2, updated_at = NOW()
		 RETURNING id, router_id, html_content, updated_at`,
		routerID, html,
	).Scan(&t.ID, &t.RouterID, &t.HTML, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ResetTemplate removes the custom template, reverting to the default embedded template
func (s *PortalTemplateStore) ResetTemplate(routerID int) error {
	_, err := s.db.Exec("DELETE FROM pisowifi_portal_templates WHERE router_id = $1", routerID)
	return err
}
