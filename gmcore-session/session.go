package gmcore_session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"
)

type Session interface {
	ID() string
	Get(key string) interface{}
	Set(key string, value interface{})
	Remove(key string)
	Has(key string) bool
	Keys() []string
	Clear()
	Destroy()
	Flash(message string)
	GetFlashes() []string
}

type Store interface {
	New(sid string) (Session, error)
	Get(sid string) (Session, error)
	Save(Session) error
	Delete(sid string) error
}

type session struct {
	id        string
	values    map[string]interface{}
	flashes   []string
	createdAt time.Time
	mu        sync.RWMutex
}

func newSession(id string) *session {
	return &session{
		id:        id,
		values:    make(map[string]interface{}),
		flashes:   make([]string, 0),
		createdAt: time.Now(),
	}
}

func (s *session) ID() string   { return s.id }
func (s *session) Get(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.values[key]
}
func (s *session) Set(key string, v interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = v
}
func (s *session) Remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
}
func (s *session) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.values[key]
	return ok
}
func (s *session) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.values))
	for k := range s.values {
		keys = append(keys, k)
	}
	return keys
}
func (s *session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = make(map[string]interface{})
}
func (s *session) Flash(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flashes = append(s.flashes, msg)
}
func (s *session) GetFlashes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.flashes
	s.flashes = make([]string, 0)
	return f
}
func (s *session) Destroy() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = make(map[string]interface{})
	s.flashes = make([]string, 0)
}

type MemoryStore struct {
	sessions map[string]*session
	mu       sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]*session)}
}

func (s *MemoryStore) New(id string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := newSession(id)
	s.sessions[id] = ns
	return ns, nil
}

func (s *MemoryStore) Get(id string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sess, ok := s.sessions[id]; ok {
		return sess, nil
	}
	return nil, nil
}

func (s *MemoryStore) Save(sess Session) error {
	if sess == nil {
		return errors.New("session cannot be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID()] = sess.(*session)
	return nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

type Manager struct {
	store    Store
	name     string
	lifetime time.Duration
}

func NewManager(store Store, name string, lifetime time.Duration) *Manager {
	return &Manager{store: store, name: name, lifetime: lifetime}
}

func (m *Manager) Name() string {
	return m.name
}

func (m *Manager) Start(w http.ResponseWriter, r *http.Request) (Session, error) {
	c, err := r.Cookie(m.name)
	var sid string
	if err != nil || c == nil || c.Value == "" {
		newSid, err := generateSid()
		if err != nil {
			return nil, err
		}
		sid = newSid
	} else {
		sid = c.Value
	}
	sess, err := m.store.Get(sid)
	if err != nil || sess == nil {
		newSid, err := generateSid()
		if err != nil {
			return nil, err
		}
		sid = newSid
		sess, err = m.store.New(sid)
		if err != nil {
			return nil, err
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.name,
		Value:    sid,
		Path:     "/",
		MaxAge:   int(m.lifetime.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
	return sess, nil
}

func generateSid() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type contextKey string

const SessionKey contextKey = "gmcore_session"

func SaveToContext(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, SessionKey, s)
}

func FromContext(ctx context.Context) Session {
	if s, ok := ctx.Value(SessionKey).(Session); ok {
		return s
	}
	return nil
}
