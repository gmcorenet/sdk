package gmcore_security

import (
	"context"
	"net/http"
	"strings"
)

type User interface {
	GetIdentifier() interface{}
	GetRoles() []string
	GetPasswordHash() string
	EraseCredentials()
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hashedPassword, plainPassword string) bool
	NeedsRehash(hashedPassword string) bool
}

type Authenticator interface {
	Authenticate(r *http.Request) (User, error)
	OnAuthSuccess(w http.ResponseWriter, r *http.Request, user User)
	OnAuthFailure(w http.ResponseWriter, r *http.Request, err error)
}

type Voter interface {
	Vote(user User, attribute string, subject interface{}) int
}

const (
	ACCESS_GRANTED = 1
	ACCESS_ABSTAIN = 0
	ACCESS_DENIED  = -1
)

type SecurityChecker struct {
	voters []Voter
}

func NewSecurityChecker() *SecurityChecker {
	return &SecurityChecker{voters: make([]Voter, 0)}
}

func (sc *SecurityChecker) AddVoter(v Voter) {
	sc.voters = append(sc.voters, v)
}

func (sc *SecurityChecker) IsGranted(user User, attribute string, subject interface{}) bool {
	for _, v := range sc.voters {
		result := v.Vote(user, attribute, subject)
		if result == ACCESS_DENIED {
			return false
		}
		if result == ACCESS_GRANTED {
			return true
		}
	}
	return false
}

type RoleVoter struct{}

func (v *RoleVoter) Vote(user User, attribute string, subject interface{}) int {
	if user == nil {
		return ACCESS_DENIED
	}
	for _, role := range user.GetRoles() {
		if strings.EqualFold(role, "ROLE_"+attribute) || strings.EqualFold(role, attribute) {
			return ACCESS_GRANTED
		}
	}
	return ACCESS_DENIED
}

type contextKey string

const UserKey contextKey = "gmcore_security_user"

func SaveUserToContext(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

func UserFromContext(ctx context.Context) User {
	if u, ok := ctx.Value(UserKey).(User); ok {
		return u
	}
	return nil
}

type SimpleUser struct {
	identifier    interface{}
	roles        []string
	passwordHash string
}

func NewSimpleUser(identifier interface{}, passwordHash string, roles []string) *SimpleUser {
	return &SimpleUser{
		identifier:    identifier,
		passwordHash: passwordHash,
		roles:        roles,
	}
}

func (u *SimpleUser) GetIdentifier() interface{} { return u.identifier }
func (u *SimpleUser) GetRoles() []string        { return u.roles }
func (u *SimpleUser) GetPasswordHash() string   { return u.passwordHash }
func (u *SimpleUser) EraseCredentials()        { u.passwordHash = "" }

func (u *SimpleUser) SetPasswordHash(hash string) { u.passwordHash = hash }
