package gmcore_settings

import (
	"context"
	"os"
	"strings"
	"sync"
)

type Encryptor interface {
	Encrypt(value string) (string, error)
	Decrypt(value string) (string, error)
}

type Config struct {
	DSN       string
	Encryptor Encryptor
}

func OpenWithConfig(ctx context.Context, cfg Config) (Store, error) {
	return &memoryStore{items: make(map[string]StoreItem)}, nil
}

type Setting struct {
	Key   string
	Value interface{}
	Type  string
}

type Settings interface {
	Get(key string) interface{}
	Set(key string, value interface{})
	Has(key string) bool
	Remove(key string)
	All() map[string]interface{}
}

type settings struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

func New() *settings {
	return &settings{data: make(map[string]interface{})}
}

func (s *settings) Get(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

func (s *settings) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *settings) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[key]
	return ok
}

func (s *settings) Remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *settings) All() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range s.data {
		result[k] = v
	}
	return result
}

func (s *settings) GetString(key string, defaultVal string) string {
	if v, ok := s.Get(key).(string); ok {
		return v
	}
	return defaultVal
}

func (s *settings) GetInt(key string, defaultVal int) int {
	if v, ok := s.Get(key).(int); ok {
		return v
	}
	return defaultVal
}

func (s *settings) GetBool(key string, defaultVal bool) bool {
	if v, ok := s.Get(key).(bool); ok {
		return v
	}
	return defaultVal
}

func (s *settings) GetFloat(key string, defaultVal float64) float64 {
	if v, ok := s.Get(key).(float64); ok {
		return v
	}
	return defaultVal
}

func (s *settings) GetStrings(key string, defaultVal []string) []string {
	if v, ok := s.Get(key).([]string); ok {
		return v
	}
	return defaultVal
}

type Manager struct {
	settings map[string]Settings
	mu       sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{settings: make(map[string]Settings)}
}

func (m *Manager) GetSettings(namespace string) Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.settings[namespace]; ok {
		return s
	}
	return New()
}

func (m *Manager) AddSettings(namespace string, settings Settings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[namespace] = settings
}

func (m *Manager) AllSettings(namespace string) map[string]interface{} {
	s := m.GetSettings(namespace)
	return s.All()
}

type EnvironmentSettings struct {
	*settings
	envPrefix string
}

func NewEnvironmentSettings(prefix string) *EnvironmentSettings {
	return &EnvironmentSettings{
		settings:  New(),
		envPrefix: prefix,
	}
}

func (s *EnvironmentSettings) LoadFromEnv() {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key := pair[0]
		value := pair[1]
		if s.envPrefix != "" && !strings.HasPrefix(key, s.envPrefix) {
			continue
		}
		envKey := key
		if s.envPrefix != "" {
			envKey = strings.TrimPrefix(key, s.envPrefix+"_")
			envKey = strings.ReplaceAll(envKey, "_", ".")
		}
		s.Set(envKey, value)
	}
}

func (s *EnvironmentSettings) SetPrefix(prefix string) {
	s.envPrefix = prefix
}

type StoreItem struct {
	Key         string
	Value       interface{}
	Type        string
	Description string
	Editable    bool
	Encrypted   bool
}

type Store interface {
	List() []StoreItem
	Get(key string) (StoreItem, bool)
	SetWithOptions(ctx context.Context, key, value, valueType, description string, editable, encrypted bool) error
}

type memoryStore struct {
	items map[string]StoreItem
	mu    sync.RWMutex
}

func NewStore() Store {
	return &memoryStore{items: make(map[string]StoreItem)}
}

func (s *memoryStore) List() []StoreItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]StoreItem, 0, len(s.items))
	for _, item := range s.items {
		result = append(result, item)
	}
	return result
}

func (s *memoryStore) Get(key string) (StoreItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[key]
	return item, ok
}

func (s *memoryStore) SetWithOptions(ctx context.Context, key, value, valueType, description string, editable, encrypted bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = StoreItem{
		Key:         key,
		Value:       value,
		Type:        valueType,
		Description: description,
		Editable:    editable,
		Encrypted:   encrypted,
	}
	return nil
}
