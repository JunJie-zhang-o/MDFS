package auth

import (
	"testing"
	"time"

	"mdfs/internal/config"
)

func TestLoginAndSessionExpiry(t *testing.T) {
	manager := New(config.Config{Server: config.Server{SessionTimeout: "1m"}, Users: []config.User{{Name: "admin", Password: "secret", Permissions: "RWD"}}})
	now := time.Unix(1000, 0)
	manager.now = func() time.Time { return now }
	token, principal, ok := manager.Login("admin", "secret")
	if !ok || !principal.Policy.For("/").Delete {
		t.Fatal("login failed")
	}
	now = now.Add(30 * time.Second)
	if !manager.FromToken(token).Authenticated {
		t.Fatal("session expired too early")
	}
	now = now.Add(2 * time.Minute)
	if manager.FromToken(token).Authenticated {
		t.Fatal("session did not expire")
	}
}
