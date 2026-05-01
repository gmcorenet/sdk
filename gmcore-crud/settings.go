package gmcorecrud

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	gmcoresettings "gmcore-settings"
)

type SettingsBackendConfig struct {
	Store *gmcoresettings.Store
}

type SettingsBackend struct {
	store *gmcoresettings.Store
}

func NewSettingsBackend(cfg SettingsBackendConfig) (*SettingsBackend, error) {
	if cfg.Store == nil {
		return nil, errors.New("missing settings store")
	}
	return &SettingsBackend{store: cfg.Store}, nil
}

func (b *SettingsBackend) Kind() BackendKind { return BackendSettings }

func (b *SettingsBackend) List(ctx context.Context, cfg Config, params ListParams) ([]Record, error) {
	_ = ctx
	items := b.store.List()
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	records := make([]Record, 0, len(items))
	search := strings.ToLower(strings.TrimSpace(params.Search))
	for _, item := range items {
		if search != "" && !strings.Contains(strings.ToLower(item.Key), search) && !strings.Contains(strings.ToLower(item.Value), search) {
			continue
		}
		records = append(records, Record{
			"key":         item.Key,
			"value":       item.Value,
			"type":        item.Type,
			"description": item.Description,
			"editable":    item.Editable,
			"encrypted":   item.Encrypted,
		})
	}
	start := params.Offset
	if start > len(records) {
		start = len(records)
	}
	end := start + params.Limit
	if params.Limit <= 0 || end > len(records) {
		end = len(records)
	}
	return records[start:end], nil
}

func (b *SettingsBackend) Count(ctx context.Context, cfg Config, params ListParams) (int, error) {
	items, err := b.List(ctx, cfg, ListParams{Search: params.Search, Limit: 0, Offset: 0})
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (b *SettingsBackend) Get(ctx context.Context, cfg Config, key string, scope map[string]interface{}) (Record, error) {
	_ = ctx
	_ = cfg
	_ = scope
	current, ok := b.store.Get(key)
	if !ok {
		return nil, errors.New("not found")
	}
	return Record{
		"key":         current.Key,
		"value":       current.Value,
		"type":        current.Type,
		"description": current.Description,
		"editable":    current.Editable,
		"encrypted":   current.Encrypted,
	}, nil
}

func (b *SettingsBackend) Create(ctx context.Context, cfg Config, record Record, scope map[string]interface{}) (Record, error) {
	return b.save(ctx, record)
}

func (b *SettingsBackend) Update(ctx context.Context, cfg Config, key string, record Record, scope map[string]interface{}) (Record, error) {
	record["key"] = key
	return b.save(ctx, record)
}

func (b *SettingsBackend) Delete(ctx context.Context, cfg Config, key string, scope map[string]interface{}) error {
	return errors.New("delete not supported for settings")
}

func (b *SettingsBackend) Bulk(ctx context.Context, cfg Config, action string, keys []string, scope map[string]interface{}) error {
	return errors.New("bulk not supported for settings")
}

func (b *SettingsBackend) save(ctx context.Context, record Record) (Record, error) {
	key := strings.TrimSpace(stringify(record["key"]))
	if key == "" {
		return nil, errors.New("missing setting key")
	}
	value := stringify(record["value"])
	valueType := strings.TrimSpace(stringify(record["type"]))
	if valueType == "" {
		valueType = "string"
	}
	description := stringify(record["description"])
	editable := parseBool(record["editable"])
	encrypted := parseBool(record["encrypted"])
	if err := b.store.SetWithOptions(ctx, key, value, valueType, description, editable, encrypted); err != nil {
		return nil, err
	}
	return b.Get(ctx, Config{}, key, nil)
}

func stringify(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func parseBool(value interface{}) bool {
	return strings.EqualFold(strings.TrimSpace(stringify(value)), "true") ||
		strings.EqualFold(strings.TrimSpace(stringify(value)), "on") ||
		strings.TrimSpace(stringify(value)) == "1" ||
		strings.EqualFold(strings.TrimSpace(stringify(value)), "yes")
}
