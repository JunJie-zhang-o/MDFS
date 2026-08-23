package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"mdfs/internal/access"
	"mdfs/internal/config"
)

const CookieName = "mdfs_session"

type Principal struct {
	Name          string
	Authenticated bool
	Policy        access.Policy
}

type user struct {
	name     string
	password string
	policy   access.Policy
}

type session struct {
	username string
	lastSeen time.Time
}

type Manager struct {
	mu        sync.Mutex
	users     map[string]user
	anonymous Principal
	sessions  map[string]session
	timeout   time.Duration
	now       func() time.Time
}

func New(cfg config.Config) *Manager {
	manager := &Manager{
		users:    make(map[string]user, len(cfg.Users)),
		sessions: make(map[string]session),
		timeout:  cfg.SessionDuration(),
		now:      time.Now,
	}
	manager.anonymous = Principal{Policy: policy(cfg.Anonymous.Permissions, cfg.Anonymous.Rules)}
	for _, configured := range cfg.Users {
		manager.users[strings.ToLower(configured.Name)] = user{
			name: configured.Name, password: configured.Password,
			policy: policy(configured.Permissions, configured.Rules),
		}
	}
	return manager
}

func policy(defaultPermissions string, configured []config.Rule) access.Policy {
	rules := make([]access.Rule, 0, len(configured))
	for _, rule := range configured {
		rules = append(rules, access.Rule{Pattern: rule.Pattern, Permissions: access.Parse(rule.Permissions)})
	}
	return access.Policy{Default: access.Parse(defaultPermissions), Rules: rules}
}

func (m *Manager) Login(username, password string) (string, Principal, bool) {
	configured, ok := m.users[strings.ToLower(username)]
	if !ok || !equal(configured.password, password) {
		return "", Principal{}, false
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", Principal{}, false
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	m.mu.Lock()
	m.sessions[token] = session{username: strings.ToLower(username), lastSeen: m.now()}
	m.mu.Unlock()
	return token, Principal{Name: configured.name, Authenticated: true, Policy: configured.policy}, true
}

func (m *Manager) Logout(token string) {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}

func (m *Manager) FromRequest(request *http.Request) Principal {
	cookie, err := request.Cookie(CookieName)
	if err != nil {
		return m.anonymous
	}
	return m.FromToken(cookie.Value)
}

func (m *Manager) FromToken(token string) Principal {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.sessions[token]
	if !ok || m.now().Sub(current.lastSeen) > m.timeout {
		delete(m.sessions, token)
		return m.anonymous
	}
	configured, ok := m.users[current.username]
	if !ok {
		delete(m.sessions, token)
		return m.anonymous
	}
	current.lastSeen = m.now()
	m.sessions[token] = current
	return Principal{Name: configured.name, Authenticated: true, Policy: configured.policy}
}

func (m *Manager) Basic(username, password string) (Principal, bool) {
	configured, ok := m.users[strings.ToLower(username)]
	if !ok || !equal(configured.password, password) {
		return Principal{}, false
	}
	return Principal{Name: configured.name, Authenticated: true, Policy: configured.policy}, true
}

func (m *Manager) Anonymous() Principal { return m.anonymous }

func equal(expected, actual string) bool {
	expectedHash := sha256.Sum256([]byte(expected))
	actualHash := sha256.Sum256([]byte(actual))
	return subtle.ConstantTimeCompare(expectedHash[:], actualHash[:]) == 1
}
