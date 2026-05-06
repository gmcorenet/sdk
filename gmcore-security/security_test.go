package gmcore_security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBCryptHasher_Hash(t *testing.T) {
	h := NewBCryptHasher(10)
	hash, err := h.Hash("password123")
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if hash == "password123" {
		t.Error("hash should not equal plain password")
	}
}

func TestBCryptHasher_Verify(t *testing.T) {
	h := NewBCryptHasher(10)
	hash, _ := h.Hash("password123")

	if !h.Verify(hash, "password123") {
		t.Error("Verify should return true for correct password")
	}
	if h.Verify(hash, "wrongpassword") {
		t.Error("Verify should return false for wrong password")
	}
}

func TestBCryptHasher_NeedsRehash(t *testing.T) {
	h := NewBCryptHasher(12)
	hash, _ := h.Hash("password123")

	if h.NeedsRehash(hash) {
		t.Error("freshly hashed password should not need rehash")
	}

	lowCostHasher := NewBCryptHasher(4)
	lowCostHash, _ := lowCostHasher.Hash("password123")

	if !h.NeedsRehash(lowCostHash) {
		t.Error("password with lower cost should need rehash")
	}
}

func TestSimplePasswordHasher_Hash(t *testing.T) {
	s := NewSimplePasswordHasher()
	hash, err := s.Hash("mypassword")
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestSimplePasswordHasher_Hash_Empty(t *testing.T) {
	s := NewSimplePasswordHasher()
	_, err := s.Hash("")
	if err == nil {
		t.Error("expected error for empty password")
	}
}

func TestSimplePasswordHasher_Verify(t *testing.T) {
	s := NewSimplePasswordHasher()
	hash, _ := s.Hash("mypassword")

	if !s.Verify(hash, "mypassword") {
		t.Error("Verify should return true for correct password")
	}
	if s.Verify(hash, "wrong") {
		t.Error("Verify should return false for wrong password")
	}
}

func TestSimplePasswordHasher_NeedsRehash(t *testing.T) {
	s := NewSimplePasswordHasher()
	hash, _ := s.Hash("password")
	_ = hash
}

func TestSimpleUser_GetRoles(t *testing.T) {
	u := NewSimpleUser("alice", "hash", []string{"ROLE_ADMIN", "ROLE_USER"})
	roles := u.GetRoles()
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

func TestSimpleUser_GetPassword(t *testing.T) {
	u := NewSimpleUser("alice", "secret_hash", []string{"ROLE_USER"})
	if u.GetPassword() != "secret_hash" {
		t.Error("GetPassword should return stored hash")
	}
}

func TestSimpleUser_EraseCredentials(t *testing.T) {
	u := NewSimpleUser("alice", "secret_hash", []string{"ROLE_USER"})
	u.EraseCredentials()
	if u.GetPassword() != "" {
		t.Error("password should be empty after EraseCredentials")
	}
}

func TestSecurityChecker_AddVoter(t *testing.T) {
	c := NewSecurityChecker()
	c.AddVoter(NewRoleVoter("ROLE_"))
	if len(c.voters) != 1 {
		t.Error("expected 1 voter")
	}
}

func TestSecurityChecker_IsGranted_NoVoters(t *testing.T) {
	c := NewSecurityChecker()
	u := NewSimpleUser("alice", "hash", []string{"ROLE_USER"})
	if c.IsGranted(u, "SOME_ATTRIBUTE", nil) {
		t.Error("should be false when no voters grant access")
	}
}

func TestSecurityChecker_IsGranted_Denied(t *testing.T) {
	c := NewSecurityChecker()
	voter := &denyVoter{}
	c.AddVoter(voter)
	u := NewSimpleUser("alice", "hash", []string{"ROLE_USER"})
	if c.IsGranted(u, "DENY_ME", nil) {
		t.Error("should be denied")
	}
}

func TestSecurityChecker_IsGranted_Granted(t *testing.T) {
	c := NewSecurityChecker()
	c.AddVoter(NewRoleVoter("ROLE_"))
	u := NewSimpleUser("alice", "hash", []string{"ROLE_ADMIN"})
	if !c.IsGranted(u, "ADMIN", nil) {
		t.Error("should be granted")
	}
}

type denyVoter struct{}

func (v *denyVoter) Vote(user User, attribute string, subject interface{}) int {
	return ACCESS_DENIED
}

func TestRoleVoter_Vote_Granted(t *testing.T) {
	v := NewRoleVoter("ROLE_")
	u := NewSimpleUser("alice", "hash", []string{"ROLE_ADMIN"})
	if v.Vote(u, "ADMIN", nil) != ACCESS_GRANTED {
		t.Error("expected ACCESS_GRANTED")
	}
}

func TestRoleVoter_Vote_Abstain(t *testing.T) {
	v := NewRoleVoter("ROLE_")
	u := NewSimpleUser("alice", "hash", []string{"ROLE_USER"})
	if v.Vote(u, "ADMIN", nil) != ACCESS_ABSTAIN {
		t.Error("expected ACCESS_ABSTAIN")
	}
}

func TestRoleVoter_Vote_NilUser(t *testing.T) {
	v := NewRoleVoter("ROLE_")
	if v.Vote(nil, "ADMIN", nil) != ACCESS_DENIED {
		t.Error("nil user should be ACCESS_DENIED")
	}
}

func TestRoleVoter_DefaultPrefix(t *testing.T) {
	v := NewRoleVoter("")
	if v.rolePrefix != "ROLE_" {
		t.Errorf("expected ROLE_, got %s", v.rolePrefix)
	}
}

func TestBasicAuthenticator_AddUser(t *testing.T) {
	a := NewBasicAuthenticator("test", NewBCryptHasher(10))
	a.AddUser("alice", "hash123")
	if a.Users["alice"] != "hash123" {
		t.Error("user not added correctly")
	}
}

func TestBasicAuthenticator_Authenticate_Missing(t *testing.T) {
	a := NewBasicAuthenticator("test", NewBCryptHasher(10))
	req := httptest.NewRequest("GET", "/", nil)
	_, err := a.Authenticate(req)
	if err != ErrCredentialsMissing {
		t.Errorf("expected ErrCredentialsMissing, got %v", err)
	}
}

func TestBasicAuthenticator_Authenticate_UserNotFound(t *testing.T) {
	a := NewBasicAuthenticator("test", NewBCryptHasher(10))
	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("alice", "password")
	_, err := a.Authenticate(req)
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestBasicAuthenticator_Authenticate_InvalidPassword(t *testing.T) {
	h := NewBCryptHasher(10)
	hash, _ := h.Hash("correctpassword")
	a := NewBasicAuthenticator("test", h)
	a.AddUser("alice", hash)

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("alice", "wrongpassword")
	_, err := a.Authenticate(req)
	if err != ErrInvalidPassword {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestBasicAuthenticator_Authenticate_Success(t *testing.T) {
	h := NewBCryptHasher(10)
	hash, _ := h.Hash("correctpassword")
	a := NewBasicAuthenticator("test", h)
	a.AddUser("alice", hash)

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("alice", "correctpassword")
	user, err := a.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected user")
	}
	if user.GetRoles()[0] != "ROLE_USER" {
		t.Error("expected default ROLE_USER")
	}
}

func TestBasicAuthenticator_OnAuthSuccess(t *testing.T) {
	a := NewBasicAuthenticator("test", NewBCryptHasher(10))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	a.OnAuthSuccess(w, r, nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", w.Code)
	}
}

func TestBasicAuthenticator_OnAuthFailure(t *testing.T) {
	a := NewBasicAuthenticator("test", NewBCryptHasher(10))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	a.OnAuthFailure(w, r, ErrInvalidPassword)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status Unauthorized, got %d", w.Code)
	}
	authHeader := w.Header().Get("WWW-Authenticate")
	if authHeader == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestSecurityError_Error(t *testing.T) {
	e := &SecurityError{Message: "test message"}
	if e.Error() != "test message" {
		t.Errorf("expected 'test message', got %q", e.Error())
	}
}

func TestAuthChecker_IsGranted(t *testing.T) {
	c := NewAuthChecker()
	c.AddVoter(NewRoleVoter("ROLE_"))
	u := NewSimpleUser("alice", "hash", []string{"ROLE_ADMIN"})
	if !c.IsGranted(u, "ADMIN", nil) {
		t.Error("should be granted")
	}
}

func TestAccessConstants(t *testing.T) {
	if ACCESS_GRANTED != 1 {
		t.Errorf("ACCESS_GRANTED should be 1, got %d", ACCESS_GRANTED)
	}
	if ACCESS_DENIED != -1 {
		t.Errorf("ACCESS_DENIED should be -1, got %d", ACCESS_DENIED)
	}
	if ACCESS_ABSTAIN != 0 {
		t.Errorf("ACCESS_ABSTAIN should be 0, got %d", ACCESS_ABSTAIN)
	}
}
