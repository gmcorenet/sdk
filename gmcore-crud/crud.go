package gmcore_crud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isValidIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return sqlIdentifierPattern.MatchString(value)
}

type PrimaryKeyType string

const (
	PrimaryKeyUUID PrimaryKeyType = "uuid"
	PrimaryKeyInt  PrimaryKeyType = "int"
)

type BackendKind string

const (
	BackendDatabase BackendKind = "database"
	BackendSettings BackendKind = "settings"
	BackendText     BackendKind = "text"
	BackendCSV      BackendKind = "csv"
	BackendArray    BackendKind = "array"
)

type Operation string

const (
	OpList   Operation = "list"
	OpGet    Operation = "get"
	OpCreate Operation = "create"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
	OpBulk   Operation = "bulk"
)

type HookStage string

const (
	HookPre  HookStage = "pre"
	HookPost HookStage = "post"
)

type Record map[string]interface{}

type FilterOperator string

const (
	FilterOperatorEq     FilterOperator = "eq"
	FilterOperatorNeq    FilterOperator = "neq"
	FilterOperatorLike   FilterOperator = "like"
	FilterOperatorGt     FilterOperator = "gt"
	FilterOperatorGte    FilterOperator = "gte"
	FilterOperatorLt     FilterOperator = "lt"
	FilterOperatorLte    FilterOperator = "lte"
	FilterOperatorIn     FilterOperator = "in"
	FilterOperatorIsNull FilterOperator = "is_null"
)

type ViewMode string

const (
	ViewModeTable ViewMode = "table"
)

type ValueFormatter func(Record) string

type ColumnFilter struct {
	Field    string         `json:"field"`
	Operator FilterOperator `json:"operator"`
	Value    string         `json:"value"`
	Values   []string       `json:"values"`
}

type ListParams struct {
	Page          int
	PerPage       int
	Limit         int
	Offset        int
	Search        string
	Filters       map[string]string
	ColumnFilters []ColumnFilter
	Sort          []string
	View          ViewMode
	Scope         map[string]interface{}
	Select        []string
	Preload       []string
}

type Field struct {
	Name            string           `json:"name"`
	Label           string           `json:"label"`
	LabelKey        string           `json:"label_key"`
	Type            string           `json:"type"`
	Required        bool             `json:"required"`
	Editable        bool             `json:"editable"`
	Filterable      bool             `json:"filterable"`
	Sortable        bool             `json:"sortable"`
	Searchable      bool             `json:"searchable"`
	Primary         bool             `json:"primary"`
	Relation        string           `json:"relation"`
	ValueField      string           `json:"value_field"`
	DisplayField    string           `json:"display_field"`
	Default         interface{}      `json:"default"`
	Options        []interface{}     `json:"options"`
	OptionLabels   map[string]string `json:"option_labels"`
	Validator      string           `json:"validator"`
	Formatter      ValueFormatter   `json:"-"`
	Widget         string           `json:"widget"`
	Multiple       bool             `json:"multiple"`
	Min            *float64         `json:"min"`
	Max            *float64         `json:"max"`
	Step           *float64         `json:"step"`
	Placeholder    string           `json:"placeholder"`
	HelpText       string           `json:"help_text"`
	HelpTextKey    string           `json:"help_text_key"`
	HelpKey        string           `json:"help_key"`
	Writable       bool             `json:"writable"`
	Visible        bool             `json:"visible"`
	Autocomplete   string           `json:"autocomplete"`
}

type RelationType string

const (
	RelationHasOne     RelationType = "has_one"
	RelationHasMany    RelationType = "has_many"
	RelationBelongsTo  RelationType = "belongs_to"
	RelationMorphOne   RelationType = "morph_one"
	RelationMorphMany  RelationType = "morph_many"
	RelationManyToMany RelationType = "many_to_many"
)

type Relation struct {
	Name                       string       `json:"name"`
	Type                       RelationType `json:"type"`
	Model                      string       `json:"model"`
	LocalField                 string       `json:"local_field"`
	ForeignKey                 string       `json:"foreign_key"`
	ValueField                 string       `json:"value_field"`
	DisplayField               string       `json:"display_field"`
	Scope                      string       `json:"scope"`
	TargetTable                string       `json:"target_table"`
	TargetPrimaryKey           string       `json:"target_primary_key"`
	TargetSchema               string       `json:"target_schema"`
	PivotTable                 string       `json:"pivot_table"`
	PivotLocalKey              string       `json:"pivot_local_key"`
	PivotForeignKey            string       `json:"pivot_foreign_key"`
	PropagateLocalValueChange  bool         `json:"propagate_local_value_change"`
	SyncSelectionAssign        bool         `json:"sync_selection_assign"`
	OnDelete                   string       `json:"on_delete"`
	Async                      bool         `json:"async"`
	AsyncDebounce              int          `json:"async_debounce"`
	AsyncLimit                 int          `json:"async_limit"`
	Widget                     string       `json:"widget"`
}

type RelationOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type RelationOptionsResult struct {
	Relation string           `json:"relation"`
	Options  []RelationOption `json:"options"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
	Total    int              `json:"total"`
	HasMore  bool             `json:"has_more"`
}

type ManyToManyRelation struct {
	Relation      Relation
	PivotTable    string
	PivotLocalKey string
	PivotOtherKey string
}

type FilterPreset struct {
	Name    string            `json:"name"`
	Label   string            `json:"label"`
	Filters map[string]string `json:"filters"`
}

type IndexColumn struct {
	Field            string            `json:"field"`
	Label            string            `json:"label"`
	Unique           bool              `json:"unique"`
	Primary          bool              `json:"primary"`
	DisplayFormatter func(Record) string `json:"-"`
	DisplayTemplate  string            `json:"display_template"`
	Formatter        func(Record) string `json:"-"`
}

type IndexDefinition struct {
	Columns []IndexColumn `json:"columns"`
}

type Action struct {
	Name            string                 `json:"name"`
	Label           string                 `json:"label"`
	LabelKey        string                 `json:"label_key"`
	Action          Operation              `json:"action"`
	Fields          []Field                `json:"fields"`
	Scope           string                 `json:"scope"`
	Confirm         bool                   `json:"confirm"`
	ConfirmText     string                 `json:"confirm_text"`
	Redirect        string                 `json:"redirect"`
	RedirectUrl     string                 `json:"redirect_url"`
	Response        map[string]interface{} `json:"response"`
	Metadata        map[string]interface{} `json:"metadata"`
	Visible         func(Record) bool      `json:"-"`
	VisibleWhen     string                 `json:"visible_when"`
	URLTemplate     string                 `json:"url_template"`
	Target          string                 `json:"target"`
	Icon            string                 `json:"icon"`
	Order           int                    `json:"order"`
	Group           string                 `json:"group"`
	SeparatorBefore bool                   `json:"separator_before"`
	SeparatorAfter  bool                   `json:"separator_after"`
	URL             string                 `json:"url"`
	GroupLabel      string                 `json:"group_label"`
}

type ActionGroup struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

type Config struct {
	Name                 string
	Label                string
	LabelKey             string
	Backend              BackendKind
	PrimaryKey           string
	PrimaryKeyType       PrimaryKeyType
	Fields               []Field
	Relations            []Relation
	Actions              []Action
	FilterPresets        []FilterPreset
	Scope                string
	OrderBy              []string
	PerPage              int
	SearchableFields     []string
	FilterableFields     []string
	SortableFields       []string
	DefaultListSort      []string
	DefaultColumnFilter  []ColumnFilter
	Features             map[string]interface{}
	Hooks                map[HookStage]map[Operation][]func(Record) error
	RowActions           []Action
	BulkActions          []Action
	RedirectUrl          string
	AfterCreateRedirect  string
	AfterUpdateRedirect  string
	AfterDeleteRedirect  string
	Index                IndexDefinition `json:"index"`
	ActionGroups         []ActionGroup   `json:"action_groups"`
}

type Manager struct {
	config   Config
	backends map[BackendKind]Backend
}

func New(cfg Config, backends map[BackendKind]Backend) (*Manager, error) {
	if cfg.PrimaryKey != "id" {
		return nil, fmt.Errorf("primary key must be named id, got %q", cfg.PrimaryKey)
	}

	hasPrimaryKeyField := false
	for _, f := range cfg.Fields {
		if f.Name == cfg.PrimaryKey {
			hasPrimaryKeyField = true
			break
		}
	}
	if !hasPrimaryKeyField {
		return nil, errors.New("must define the primary key field id in the fields list")
	}

	if cfg.PrimaryKeyType != PrimaryKeyUUID && cfg.PrimaryKeyType != PrimaryKeyInt {
		return nil, errors.New("primary key type must be uuid or int")
	}

	for _, field := range cfg.Fields {
		if field.Relation != "" {
			var rel Relation
			var found bool
			for _, r := range cfg.Relations {
				if r.Name == field.Relation {
					rel = r
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("field %q references relation %q which does not exist", field.Name, field.Relation)
			}
			if rel.ValueField == "" {
				return nil, fmt.Errorf("relation %q is missing value field", rel.Name)
			}
			if rel.DisplayField == "" {
				return nil, fmt.Errorf("relation %q is missing display field", rel.Name)
			}
		}
	}

	if cfg.PerPage <= 0 {
		cfg.PerPage = 20
	}

	if cfg.Features == nil {
		cfg.Features = make(map[string]interface{})
	}

	m := &Manager{
		config:   cfg,
		backends: backends,
	}

	return m, nil
}

func (m *Manager) Backend(kind BackendKind) Backend {
	return m.backends[kind]
}

func (m *Manager) Config() Config {
	return m.config
}

func (m *Manager) Relation(name string) (Relation, bool) {
	for _, rel := range m.config.Relations {
		if rel.Name == name {
			return rel, true
		}
	}
	return Relation{}, false
}

func (m *Manager) queryName() string {
	if len(m.backends) == 0 {
		return ""
	}
	for _, backend := range m.backends {
		return backend.IndexQueryName()
	}
	return ""
}

func (m *Manager) ParseListParams(request *http.Request) ListParams {
	params := ListParams{}
	if request == nil {
		return params
	}

	values := request.URL.Query()

	if page := values.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}
	if perPage := values.Get("per_page"); perPage != "" {
		if pp, err := strconv.Atoi(perPage); err == nil {
			params.PerPage = pp
		}
	}
	if limit := values.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			params.Limit = l
		}
	}
	if offset := values.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			params.Offset = o
		}
	}
	if search := values.Get("search"); search != "" {
		params.Search = search
	}
	if sort := values.Get("sort"); sort != "" {
		params.Sort = strings.Split(sort, ",")
	}

	params.Filters = make(map[string]string)
	params.ColumnFilters = make([]ColumnFilter, 0)

	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		value := vals[0]

		if strings.HasPrefix(key, "filter_") && !strings.HasSuffix(key, "_op") {
			field := strings.TrimPrefix(key, "filter_")
			opKey := "filter_" + field + "_op"
			opStr := values.Get(opKey)

			if opStr != "" {
				op := FilterOperator(opStr)
				params.ColumnFilters = append(params.ColumnFilters, ColumnFilter{
					Field:    field,
					Operator: op,
					Value:    value,
				})
			} else {
				params.Filters[field] = value
			}
		}
	}

	return params
}

func (m *Manager) ResolveRowActions(record Record) ([]Action, error) {
	groupLabelMap := make(map[string]string)
	for _, group := range m.config.ActionGroups {
		groupLabelMap[group.Name] = group.Label
	}

	var visibleActions []Action
	for _, action := range m.config.RowActions {
		if action.Visible != nil && !action.Visible(record) {
			continue
		}
		if action.VisibleWhen != "" && !evaluateVisibilityCondition(action.VisibleWhen, record) {
			continue
		}

		expanded := action
		if action.URLTemplate != "" {
			expanded.URL = expandURLTemplate(action.URLTemplate, record)
		}
		if action.Group != "" {
			expanded.GroupLabel = groupLabelMap[action.Group]
		}

		visibleActions = append(visibleActions, expanded)
	}

	sort.Slice(visibleActions, func(i, j int) bool {
		iHasGroup := visibleActions[i].Group != ""
		jHasGroup := visibleActions[j].Group != ""
		if iHasGroup != jHasGroup {
			return iHasGroup
		}
		if iHasGroup {
			return visibleActions[i].Order < visibleActions[j].Order
		}
		return visibleActions[i].Order > visibleActions[j].Order
	})

	return visibleActions, nil
}

func expandURLTemplate(template string, record Record) string {
	result := template
	for k, v := range record {
		placeholder := "<" + k + ">"
		value := ""
		if v != nil {
			value = url.PathEscape(fmt.Sprintf("%v", v))
		}
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func evaluateVisibilityCondition(condition string, record Record) bool {
	parts := strings.Split(condition, "=")
	if len(parts) != 2 {
		return true
	}
	field := strings.TrimSpace(parts[0])
	expectedValue := strings.TrimSpace(parts[1])
	actualValue, ok := record[field]
	if !ok {
		return false
	}
	return fmt.Sprintf("%v", actualValue) == expectedValue
}

type Backend interface {
	Kind() BackendKind
	IndexQueryName() string
	List(ctx context.Context, cfg Config, params ListParams) ([]Record, error)
	Get(ctx context.Context, cfg Config, key string, scope map[string]interface{}) (Record, error)
	Create(ctx context.Context, cfg Config, record Record, scope map[string]interface{}) (Record, error)
	Update(ctx context.Context, cfg Config, key string, record Record, scope map[string]interface{}) (Record, error)
	Delete(ctx context.Context, cfg Config, key string, scope map[string]interface{}) error
	Bulk(ctx context.Context, cfg Config, action string, keys []string, scope map[string]interface{}) error
}

func defaultFilterOperatorForField(cfg Config, fieldName string) FilterOperator {
	for _, f := range cfg.Fields {
		if f.Name == fieldName {
			if f.Type == "string" || f.Type == "text" {
				return FilterOperatorLike
			}
			return FilterOperatorEq
		}
	}
	return FilterOperatorEq
}

func sortFieldNames(fields []Field) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	sort.Strings(names)
	return names
}
