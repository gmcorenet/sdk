package gmcore_security

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gmcorenet/sdk/gmcore-config"
	"golang.org/x/crypto/bcrypt"
)

const (
	ACCESS_GRANTED = 1
	ACCESS_DENIED  = -1
	ACCESS_ABSTAIN = 0
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hashedPassword, plainPassword string) bool
	NeedsRehash(hashedPassword string) bool
}

type User interface {
	GetRoles() []string
	GetPassword() string
	EraseCredentials()
}

type simpleUser struct {
	username string
	password string
	roles    []string
}

func NewSimpleUser(username, password string, roles []string) User {
	return &simpleUser{
		username: username,
		password: password,
		roles:    roles,
	}
}

func (u *simpleUser) GetRoles() []string  { return u.roles }
func (u *simpleUser) GetPassword() string { return u.password }
func (u *simpleUser) EraseCredentials()   { u.password = "" }

type Voter interface {
	Vote(user User, attribute string, subject interface{}) int
}

type SecurityChecker struct {
	voters []Voter
}

func NewSecurityChecker() *SecurityChecker {
	return &SecurityChecker{voters: make([]Voter, 0)}
}

func (c *SecurityChecker) AddVoter(v Voter) {
	c.voters = append(c.voters, v)
}

func (c *SecurityChecker) IsGranted(user User, attribute string, subject interface{}) bool {
	for _, voter := range c.voters {
		result := voter.Vote(user, attribute, subject)
		if result == ACCESS_DENIED {
			return false
		}
		if result == ACCESS_GRANTED {
			return true
		}
	}
	return false
}

func (c *SecurityChecker) IsGrantedAny(user User, roles []string, subject interface{}) bool {
	for _, role := range roles {
		if c.IsGranted(user, role, subject) {
			return true
		}
	}
	return false
}

func (c *SecurityChecker) IsGrantedAll(user User, roles []string, subject interface{}) bool {
	for _, role := range roles {
		if !c.IsGranted(user, role, subject) {
			return false
		}
	}
	return len(roles) > 0
}

type Config struct {
	RolePrefix   string         `yaml:"role_prefix" json:"role_prefix"`
	DefaultRole  string         `yaml:"default_role" json:"default_role"`
	PasswordCost int            `yaml:"password_cost" json:"password_cost"`
	Firewall     FirewallConfig `yaml:"firewall" json:"firewall"`
}

type FirewallConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Patterns []string `yaml:"patterns" json:"patterns"`
	Excludes []string `yaml:"excludes" json:"excludes"`
}

func LoadConfig(appPath string) (*Config, error) {
	l := gmcore_config.NewLoader[Config](appPath)
	for _, name := range []string{"security.yaml", "security.yml"} {
		if cfg, err := l.LoadDefault(name); cfg != nil || err != nil {
			return cfg, err
		}
	}
	return nil, nil
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
	Realm  string
	Hasher PasswordHasher
	Users  map[string]string
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
	ErrUserNotFound       = &SecurityError{Message: "user_not_found"}
	ErrInvalidPassword    = &SecurityError{Message: "invalid_password"}
	ErrAccessDenied       = &SecurityError{Message: "access_denied"}
)

type SecurityError struct {
	Message string
}

func (e *SecurityError) Error() string {
	return e.Message
}
