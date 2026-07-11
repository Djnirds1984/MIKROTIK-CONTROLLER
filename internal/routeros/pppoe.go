package routeros

import (
	"mikrotik-controller/internal/models"
	"net/url"
)

// GetPPPoESecrets retrieves all PPPoE secrets (users) from the router
func (cm *ConnectionManager) GetPPPoESecrets(routerID int) ([]models.PPPoEUser, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ppp/secret")
	if err != nil {
		return nil, err
	}

	var users []models.PPPoEUser
	for _, item := range results {
		user := models.PPPoEUser{
			Name:     getString(item, "name"),
			Password: getString(item, "password"),
			Profile:  getString(item, "profile"),
			Service:  getString(item, "service"),
			Comment:  getString(item, "comment"),
			Disabled: getString(item, "disabled") == "true",
		}
		users = append(users, user)
	}

	return users, nil
}

// AddPPPoESecret adds a new PPPoE user
func (cm *ConnectionManager) AddPPPoESecret(routerID int, user models.PPPoEUser) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("name", user.Name)
	data.Set("password", user.Password)
	data.Set("profile", user.Profile)
	data.Set("service", user.Service)
	if user.Comment != "" {
		data.Set("comment", user.Comment)
	}
	if user.Disabled {
		data.Set("disabled", "true")
	}
	return restPost(conn, "/ppp/secret", data)
}

// UpdatePPPoESecret updates an existing PPPoE user
func (cm *ConnectionManager) UpdatePPPoESecret(routerID int, oldName string, user models.PPPoEUser) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("name", user.Name)
	data.Set("password", user.Password)
	data.Set("profile", user.Profile)
	data.Set("service", user.Service)
	if user.Comment != "" {
		data.Set("comment", user.Comment)
	}
	if user.Disabled {
		data.Set("disabled", "true")
	}
	return restPut(conn, "/ppp/secret/"+oldName, data)
}

// DeletePPPoESecret removes a PPPoE user
func (cm *ConnectionManager) DeletePPPoESecret(routerID int, name string) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	return restDelete(conn, "/ppp/secret/"+name)
}

// GetPPPoEProfiles retrieves all PPPoE profiles from the router
func (cm *ConnectionManager) GetPPPoEProfiles(routerID int) ([]models.PPPoEProfile, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ppp/profile")
	if err != nil {
		return nil, err
	}

	var profiles []models.PPPoEProfile
	for _, item := range results {
		profile := models.PPPoEProfile{
			Name:          getString(item, "name"),
			LocalAddress:  getString(item, "local-address"),
			RemoteAddress: getString(item, "remote-address"),
			RateLimit:     getString(item, "rate-limit"),
			ParentProfile: getString(item, "parent"),
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// AddPPPoEProfile creates a new PPPoE profile
func (cm *ConnectionManager) AddPPPoEProfile(routerID int, profile models.PPPoEProfile) error {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("name", profile.Name)
	if profile.LocalAddress != "" {
		data.Set("local-address", profile.LocalAddress)
	}
	if profile.RemoteAddress != "" {
		data.Set("remote-address", profile.RemoteAddress)
	}
	if profile.RateLimit != "" {
		data.Set("rate-limit", profile.RateLimit)
	}
	if profile.ParentProfile != "" {
		data.Set("parent", profile.ParentProfile)
	}
	return restPost(conn, "/ppp/profile", data)
}

// GetPPPoESessions retrieves active PPPoE sessions
func (cm *ConnectionManager) GetPPPoESessions(routerID int) ([]models.PPPoESession, error) {
	conn, err := cm.GetConn(routerID)
	if err != nil {
		return nil, err
	}

	results, err := restGet(conn, "/ppp/active")
	if err != nil {
		return nil, err
	}

	var sessions []models.PPPoESession
	for _, item := range results {
		session := models.PPPoESession{
			Username:  getString(item, "name"),
			Interface: getString(item, "interface"),
			Service:   getString(item, "service"),
			State:     getString(item, "state"),
			CallerID:  getString(item, "caller-id"),
			IPAddress: getString(item, "address"),
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}
