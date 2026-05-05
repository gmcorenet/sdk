package gmcore_security

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gmcorenet/sdk/gmcore-config"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	RolePrefix    string   `yaml:"role_prefix" json:"role_prefix"`
	DefaultRole   string   `yaml:"default_role" json:"default_role"`
	PasswordCost int      `yaml:"password_cost" json:"password_cost"`
	Firewall      FirewallConfig `yaml:"firewall" json:"firewall"`
}

type FirewallConfig struct {
	Enabled   bool     `yaml:"enabled" json:"enabled"`
	Patterns  []string `yaml:"patterns" json:"patterns"`
	Excludes  []string `yaml:"excludes" json:"excludes"`
}

type ConfigLoader struct {
	appPath string
	env     map[string]string
}

func NewConfigLoader(appPath string) *ConfigLoader {
	return &ConfigLoader{
		appPath: appPath,
		env:     gmcore_config.LoadAppEnv(appPath),
	}
}

func (l *ConfigLoader) Load(path string) (*Config, error) {
	cfg := &Config{}

	opts := gmcore_config.Options{
		Env:        l.env,
		Parameters: map[string]string{},
		Strict:     false,
	}

	if err := gmcore_config.LoadYAML(path, cfg, opts); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (l *ConfigLoader) LoadDefault() (*Config, error) {
	candidates := []string{
		filepath.Join(l.appPath, "config", "security.yaml"),
		filepath.Join(l.appPath, "config", "security.yml"),
		filepath.Join(l.appPath, "security.yaml"),
		filepath.Join(l.appPath, "security.yml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return l.Load(path)
		}
	}

	return nil, nil
}

func LoadConfig(appPath string) (*Config, error) {
	loader := NewConfigLoader(appPath)
	return loader.LoadDefault()
}

type BCryptHasher struct {
	cost int
}

func NewBCryptHasher(cost int) *BCryptHasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BCryptHasher{cost: cost}
}

func (h *BCryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(bytes), err
}

func (h *BCryptHasher) Verify(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

func (h *BCryptHasher) NeedsRehash(hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(""))
	if err != nil {
		cost, _ := bcrypt.Cost([]byte(hashedPassword))
		return cost < h.cost
	}
	return true
}

type SimplePasswordHasher struct {
	Cost int
}

func NewSimplePasswordHasher() *SimplePasswordHasher {
	return &SimplePasswordHasher{Cost: bcrypt.DefaultCost}
}

func (s *SimplePasswordHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.Cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *SimplePasswordHasher) Verify(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

func (s *SimplePasswordHasher) NeedsRehash(hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(""))
	return err != nil
}

type BasicAuthenticator struct {
	Realm      string
	Hasher     PasswordHasher
	Users      map[string]string
}

func NewBasicAuthenticator(realm string, hasher PasswordHasher) *BasicAuthenticator {
	return &BasicAuthenticator{
		Realm:  realm,
		Hasher: hasher,
		Users:  make(map[string]string),
	}
}

func (a *BasicAuthenticator) AddUser(username, passwordHash string) {
	a.Users[username] = passwordHash
}

func (a *BasicAuthenticator) Authenticate(r *http.Request) (User, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, ErrCredentialsMissing
	}

	hash, exists := a.Users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	if !a.Hasher.Verify(hash, password) {
		return nil, ErrInvalidPassword
	}

	return NewSimpleUser(username, hash, []string{"ROLE_USER"}), nil
}

func (a *BasicAuthenticator) OnAuthSuccess(w http.ResponseWriter, r *http.Request, user User) {
	w.WriteHeader(http.StatusOK)
}

func (a *BasicAuthenticator) OnAuthFailure(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+a.Realm+`"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

type AuthChecker struct {
	checker *SecurityChecker
}

func NewAuthChecker() *AuthChecker {
	return &AuthChecker{
		checker: NewSecurityChecker(),
	}
}

func (c *AuthChecker) AddVoter(v Voter) {
	c.checker.AddVoter(v)
}

func (c *AuthChecker) IsGranted(user User, attribute string, subject interface{}) bool {
	return c.checker.IsGranted(user, attribute, subject)
}

type RoleVoter struct {
	rolePrefix string
}

func NewRoleVoter(rolePrefix string) *RoleVoter {
	if rolePrefix == "" {
		rolePrefix = "ROLE_"
	}
	return &RoleVoter{rolePrefix: rolePrefix}
}

func (v *RoleVoter) Vote(user User, attribute string, subject interface{}) int {
	if user == nil {
		return ACCESS_DENIED
	}
	for _, role := range user.GetRoles() {
		if strings.EqualFold(role, v.rolePrefix+attribute) || strings.EqualFold(role, attribute) {
			return ACCESS_GRANTED
		}
	}
	return ACCESS_ABSTAIN
}

var (
	ErrCredentialsMissing = &SecurityError{Message: "credentials_missing"}
	ErrUserNotFound      = &SecurityError{Message: "user_not_found"}
	ErrInvalidPassword    = &SecurityError{Message: "invalid_password"}
	ErrAccessDenied      = &SecurityError{Message: "access_denied"}
)

type SecurityError struct {
	Message string
}

func (e *SecurityError) Error() string {
	return e.Message
}
