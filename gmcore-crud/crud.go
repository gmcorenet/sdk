package gmcore_crud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

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
	Searchable      bool             `json:"searchable"`
	Sortable        bool             `json:"sortable"`
	Writable        bool             `json:"writable"`
	Visible         bool             `json:"visible"`
	Filterable      bool             `json:"filterable"`
	FilterOperators []FilterOperator `json:"filter_operators"`
	Placeholder     string           `json:"placeholder"`
	HelpKey         string           `json:"help_key"`
	Relation        string           `json:"relation,omitempty"`
}

type RelationType string

const (
	RelationBelongsTo  RelationType = "belongs_to"
	RelationHasMany    RelationType = "has_many"
	RelationManyToMany RelationType = "many_to_many"
)

type Relation struct {
	Name                      string       `json:"name"`
	Type                      RelationType `json:"type"`
	TargetSchema              string       `json:"target_schema"`
	TargetTable               string       `json:"target_table"`
	TargetPrimaryKey          string       `json:"target_primary_key"`
	LocalField                string       `json:"local_field"`
	ForeignKey                string       `json:"foreign_key"`
	PivotTable                string       `json:"pivot_table"`
	PivotLocalKey             string       `json:"pivot_local_key"`
	PivotForeignKey           string       `json:"pivot_foreign_key"`
	ValueField                string       `json:"value_field"`
	DisplayField              string       `json:"display_field"`
	Widget                    string       `json:"widget"`
	Async                     bool         `json:"async"`
	AsyncLimit                int          `json:"async_limit"`
	AsyncDebounce             int          `json:"async_debounce"`
	LoadAllLimit              int          `json:"load_all_limit"`
	Placeholder               string       `json:"placeholder"`
	PropagateLocalValueChange bool         `json:"propagate_local_value_change"`
	SyncSelectionAssign       bool         `json:"sync_selection_assign"`
	OnDelete                  string       `json:"on_delete"`
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

type IndexColumn struct {
	Field            string
	Label            string
	LabelKey         string
	Sortable         bool
	Formatter        ValueFormatter
	DisplayFormatter ValueFormatter
	DisplayTemplate  string
	SortExpression   string
	Priority         int
}

type Action struct {
	Name            string
	Label           string
	Kind            string
	Method          string
	URLTemplate     string
	Target          string
	Icon            string
	Order           int
	Group           string
	Separator       bool
	SeparatorBefore bool
	SeparatorAfter  bool
	ConfirmText     string
	VisibleWhen     string
	Visible         func(Record) bool
}

type ActionGroup struct {
	Name  string
	Label string
	Order int
}

type BulkAction struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	LabelKey    string `json:"label_key"`
	Kind        string `json:"kind"`
	Method      string `json:"method"`
	ConfirmText string `json:"confirm_text"`
}

type IndexViewConfig struct {
	DefaultView ViewMode
	Allowed     []ViewMode
}

type PaginationConfig struct {
	DefaultPerPage int   `json:"default_per_page"`
	PerPageOptions []int `json:"per_page_options"`
}

type IndexConfig struct {
	TitleKey          string
	SearchPlaceholder string
	Views             IndexViewConfig
	Pagination        PaginationConfig
	Columns           []IndexColumn
}

type FormButton struct {
	Name     string
	LabelKey string
	Redirect string
	Style    string
}

type FormConfig struct {
	TitleCreateKey string
	TitleEditKey   string
	Buttons        []FormButton
}

type BreadcrumbItem struct {
	LabelKey string `json:"label_key"`
	URL      string `json:"url"`
}

type HookContext struct {
	Resource   string
	Operation  Operation
	Stage      HookStage
	Query      string
	PrimaryKey string
	Record     Record
	Selection  []string
	Scope      map[string]interface{}
}

type Hook func(context.Context, HookContext) error

type Hooks struct {
	PreList    []Hook
	PostList   []Hook
	PreGet     []Hook
	PostGet    []Hook
	PreCreate  []Hook
	PostCreate []Hook
	PreUpdate  []Hook
	PostUpdate []Hook
	PreDelete  []Hook
	PostDelete []Hook
	PreBulk    []Hook
	PostBulk   []Hook
}

type Config struct {
	Name           string
	Backend        BackendKind
	PrimaryKey     string
	PrimaryKeyType PrimaryKeyType
	SlugField      string
	SoftDelete     bool
	Fields         []Field
	Relations      []Relation
	Index          IndexConfig
	Form           FormConfig
	RowActions     []Action
	ActionGroups   []ActionGroup
	BulkActions    []BulkAction
	IconRegistry   map[string]string
	Hooks          Hooks
}

type ListResult struct {
	Records      []Record       `json:"records"`
	Total        int            `json:"total"`
	Page         int            `json:"page"`
	PerPage      int            `json:"per_page"`
	TotalPages   int            `json:"total_pages"`
	Search       string         `json:"search"`
	ColumnFilter []ColumnFilter `json:"column_filters"`
	Sort         []string       `json:"sort"`
	View         ViewMode       `json:"view"`
}

type Backend interface {
	Kind() BackendKind
	List(context.Context, Config, ListParams) ([]Record, error)
	Get(context.Context, Config, string, map[string]interface{}) (Record, error)
	Create(context.Context, Config, Record, map[string]interface{}) (Record, error)
	Update(context.Context, Config, string, Record, map[string]interface{}) (Record, error)
	Delete(context.Context, Config, string, map[string]interface{}) error
	Bulk(context.Context, Config, string, []string, map[string]interface{}) error
}

type CountableBackend interface {
	Count(context.Context, Config, ListParams) (int, error)
}

type Manager struct {
	cfg     Config
	backend Backend
}

type backendQueryNamer interface {
	IndexQueryName() string
}

func (m *Manager) Config() Config {
	return m.cfg
}

func New(cfg Config, backend Backend) (*Manager, error) {
	if strings.TrimSpace(cfg.Name) == "" {
		return nil, errors.New("missing crud name")
	}
	cfg.PrimaryKey = strings.TrimSpace(cfg.PrimaryKey)
	if cfg.PrimaryKey == "" {
		return nil, errors.New("missing primary key")
	}
	if !strings.EqualFold(cfg.PrimaryKey, "id") {
		return nil, errors.New("crud primary key must be named id")
	}
	if cfg.PrimaryKeyType == "" {
		cfg.PrimaryKeyType = PrimaryKeyUUID
	}
	if cfg.PrimaryKeyType != PrimaryKeyUUID && cfg.PrimaryKeyType != PrimaryKeyInt {
		return nil, errors.New("crud primary key type must be uuid or int")
	}
	if !hasPrimaryKeyField(cfg.Fields, cfg.PrimaryKey) {
		return nil, errors.New("crud fields must define the primary key field id")
	}
	if backend == nil {
		return nil, errors.New("missing backend")
	}
	for _, relation := range cfg.Relations {
		if strings.TrimSpace(relation.Name) == "" {
			return nil, errors.New("crud relations must have a name")
		}
		if relation.Type == "" {
			relation.Type = RelationBelongsTo
		}
		if strings.TrimSpace(relation.LocalField) == "" {
			return nil, fmt.Errorf("crud relation %q missing local field", relation.Name)
		}
		if strings.TrimSpace(relation.ValueField) == "" {
			return nil, fmt.Errorf("crud relation %q missing value field", relation.Name)
		}
		if strings.TrimSpace(relation.DisplayField) == "" {
			return nil, fmt.Errorf("crud relation %q missing display field", relation.Name)
		}
		if relation.Type == RelationManyToMany {
			if strings.TrimSpace(relation.PivotTable) == "" || strings.TrimSpace(relation.PivotLocalKey) == "" || strings.TrimSpace(relation.PivotForeignKey) == "" {
				return nil, fmt.Errorf("crud relation %q missing pivot metadata", relation.Name)
			}
		}
	}
	return &Manager{cfg: cfg, backend: backend}, nil
}

func (m *Manager) Relation(name string) (Relation, bool) {
	for _, relation := range m.cfg.Relations {
		if strings.EqualFold(strings.TrimSpace(relation.Name), strings.TrimSpace(name)) {
			return relation, true
		}
	}
	return Relation{}, false
}

func (m *Manager) queryName() string {
	if m != nil {
		if named, ok := m.backend.(backendQueryNamer); ok {
			if value := strings.TrimSpace(named.IndexQueryName()); value != "" {
				return value
			}
		}
		if value := strings.TrimSpace(m.cfg.Name); value != "" {
			return value + "_index"
		}
	}
	return "index"
}

func hasPrimaryKeyField(fields []Field, primaryKey string) bool {
	for _, field := range fields {
		if strings.EqualFold(strings.TrimSpace(field.Name), primaryKey) {
			return true
		}
	}
	return false
}

func (m *Manager) List(ctx context.Context, params ListParams) ([]Record, error) {
	params = m.NormalizeListParams(params)
	if err := runHooks(ctx, m.cfg.Hooks.PreList, HookContext{Resource: m.cfg.Name, Operation: OpList, Stage: HookPre, Query: m.queryName(), Scope: params.Scope}); err != nil {
		return nil, err
	}
	records, err := m.backend.List(ctx, m.cfg, params)
	if err != nil {
		return nil, err
	}
	if err := runHooks(ctx, m.cfg.Hooks.PostList, HookContext{Resource: m.cfg.Name, Operation: OpList, Stage: HookPost, Query: m.queryName(), Scope: params.Scope}); err != nil {
		return nil, err
	}
	return records, nil
}

func (m *Manager) ListResult(ctx context.Context, params ListParams) (ListResult, error) {
	params = m.NormalizeListParams(params)
	records, err := m.List(ctx, params)
	if err != nil {
		return ListResult{}, err
	}
	total := len(records)
	if counter, ok := m.backend.(CountableBackend); ok {
		total, err = counter.Count(ctx, m.cfg, params)
		if err != nil {
			return ListResult{}, err
		}
	}
	page := params.Page
	if page < 1 {
		page = 1
	}
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = params.Limit
	}
	totalPages := 1
	if perPage > 0 && total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	return ListResult{
		Records:      records,
		Total:        total,
		Page:         page,
		PerPage:      perPage,
		TotalPages:   totalPages,
		Search:       params.Search,
		ColumnFilter: params.ColumnFilters,
		Sort:         params.Sort,
		View:         params.View,
	}, nil
}

func (m *Manager) NormalizeListParams(params ListParams) ListParams {
	if params.Page < 1 {
		params.Page = 1
	}
	defaultPerPage := m.cfg.Index.Pagination.DefaultPerPage
	if defaultPerPage <= 0 {
		defaultPerPage = 25
	}
	if params.PerPage <= 0 {
		params.PerPage = defaultPerPage
	}
	if params.Limit <= 0 {
		params.Limit = params.PerPage
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	if params.Offset == 0 && params.Page > 1 && params.Limit > 0 {
		params.Offset = (params.Page - 1) * params.Limit
	}
	if params.View == "" {
		params.View = m.defaultView()
	}
	params.Sort = m.allowedSorts(params.Sort)
	params.ColumnFilters = m.allowedColumnFilters(params.ColumnFilters)
	for field := range params.Filters {
		if !m.isFilterableField(field) {
			delete(params.Filters, field)
		}
	}
	return params
}

func (m *Manager) defaultView() ViewMode {
	return ViewModeTable
}

func (m *Manager) ParseListParams(req *http.Request) ListParams {
	query := req.URL.Query()
	params := ListParams{
		Page:          intQuery(query, "page", 1),
		PerPage:       intQuery(query, "per_page", m.cfg.Index.Pagination.DefaultPerPage),
		Search:        strings.TrimSpace(query.Get("q")),
		Sort:          parseSort(query.Get("sort")),
		View:          ViewMode(strings.TrimSpace(query.Get("view"))),
		Filters:       map[string]string{},
		ColumnFilters: parseColumnFilters(query),
	}
	for _, field := range m.cfg.Fields {
		value := strings.TrimSpace(query.Get("filter_" + field.Name))
		if value != "" {
			operator := FilterOperator(strings.TrimSpace(query.Get("filter_" + field.Name + "_op")))
			if operator != "" {
				params.ColumnFilters = append(params.ColumnFilters, ColumnFilter{
					Field:    field.Name,
					Operator: operator,
					Value:    value,
				})
				continue
			}
			params.Filters[field.Name] = value
		}
	}
	return m.NormalizeListParams(params)
}

func (m *Manager) allowedSorts(input []string) []string {
	out := make([]string, 0, len(input))
	for _, current := range input {
		current = strings.TrimSpace(current)
		if current == "" {
			continue
		}
		desc := strings.HasPrefix(current, "-")
		name := strings.TrimPrefix(current, "-")
		expr, ok := m.sortExpressionFor(name)
		if !ok {
			continue
		}
		if desc {
			expr = "-" + expr
		}
		out = append(out, expr)
	}
	return out
}

func (m *Manager) sortExpressionFor(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	for _, column := range m.cfg.Index.Columns {
		if !column.Sortable || !strings.EqualFold(strings.TrimSpace(column.Field), name) {
			continue
		}
		if value := strings.TrimSpace(column.SortExpression); value != "" {
			return value, true
		}
		return strings.TrimSpace(column.Field), true
	}
	for _, field := range m.cfg.Fields {
		if !field.Sortable || !strings.EqualFold(strings.TrimSpace(field.Name), name) {
			continue
		}
		return strings.TrimSpace(field.Name), true
	}
	return "", false
}

func (m *Manager) allowedColumnFilters(input []ColumnFilter) []ColumnFilter {
	out := make([]ColumnFilter, 0, len(input))
	for _, filter := range input {
		if !m.isFilterableField(filter.Field) {
			continue
		}
		out = append(out, filter)
	}
	return out
}

func defaultFilterOperatorForField(cfg Config, fieldName string) FilterOperator {
	fieldName = strings.TrimSpace(fieldName)
	for _, field := range cfg.Fields {
		if !strings.EqualFold(strings.TrimSpace(field.Name), fieldName) {
			continue
		}
		fieldType := strings.ToLower(strings.TrimSpace(field.Type))
		if fieldType == "string" || fieldType == "text" || fieldType == "varchar" {
			return FilterOperatorLike
		}
		return FilterOperatorEq
	}
	return FilterOperatorEq
}

func (m *Manager) isFilterableField(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, field := range m.cfg.Fields {
		if field.Filterable && strings.EqualFold(strings.TrimSpace(field.Name), name) {
			return true
		}
	}
	for _, column := range m.cfg.Index.Columns {
		if strings.EqualFold(strings.TrimSpace(column.Field), name) {
			return false
		}
	}
	return false
}

func (m *Manager) IndexModel(ctx context.Context, params ListParams) (IndexModel, error) {
	result, err := m.ListResult(ctx, params)
	if err != nil {
		return IndexModel{}, err
	}
	actions := make(map[string][]ResolvedAction, len(result.Records))
	for _, record := range result.Records {
		key := fmt.Sprint(record[m.cfg.PrimaryKey])
		actions[key], err = m.ResolveRowActions(record)
		if err != nil {
			return IndexModel{}, err
		}
	}
	return IndexModel{
		Resource:     m.cfg.Name,
		TitleKey:     m.cfg.Index.TitleKey,
		View:         ViewModeTable,
		AllowedViews: []ViewMode{ViewModeTable},
		Fields:       m.cfg.Fields,
		Columns:      m.resolveColumns(),
		Rows:         m.resolveRows(result.Records, actions),
		Result:       result,
		BulkActions:  m.cfg.BulkActions,
		Breadcrumbs: []BreadcrumbItem{
			{LabelKey: "gmcore.dashboard", URL: "/"},
			{LabelKey: m.cfg.Index.TitleKey, URL: ""},
		},
		SearchPlaceholder: m.cfg.Index.SearchPlaceholder,
		Pagination:        m.cfg.Index.Pagination,
		FilterFields:      m.filterFields(),
	}, nil
}

func (m *Manager) FormModel(mode string, record Record) FormModel {
	titleKey := m.cfg.Form.TitleCreateKey
	if mode == "edit" {
		titleKey = m.cfg.Form.TitleEditKey
	}
	return FormModel{
		Resource:    m.cfg.Name,
		Mode:        mode,
		TitleKey:    titleKey,
		Fields:      m.cfg.Fields,
		Record:      record,
		Buttons:     normalizeFormButtons(m.cfg.Form.Buttons),
		Breadcrumbs: formBreadcrumbs(m.cfg, mode, record),
	}
}

func (m *Manager) Get(ctx context.Context, key string, scope map[string]interface{}) (Record, error) {
	if err := runHooks(ctx, m.cfg.Hooks.PreGet, HookContext{Resource: m.cfg.Name, Operation: OpGet, Stage: HookPre, PrimaryKey: key, Query: m.queryName(), Scope: scope}); err != nil {
		return nil, err
	}
	record, err := m.backend.Get(ctx, m.cfg, key, scope)
	if err != nil {
		return nil, err
	}
	if err := runHooks(ctx, m.cfg.Hooks.PostGet, HookContext{Resource: m.cfg.Name, Operation: OpGet, Stage: HookPost, PrimaryKey: key, Record: record, Query: m.queryName(), Scope: scope}); err != nil {
		return nil, err
	}
	return record, nil
}

func (m *Manager) Create(ctx context.Context, record Record, scope map[string]interface{}) (Record, error) {
	if err := runHooks(ctx, m.cfg.Hooks.PreCreate, HookContext{Resource: m.cfg.Name, Operation: OpCreate, Stage: HookPre, Record: record, Scope: scope}); err != nil {
		return nil, err
	}
	created, err := m.backend.Create(ctx, m.cfg, record, scope)
	if err != nil {
		return nil, err
	}
	if err := runHooks(ctx, m.cfg.Hooks.PostCreate, HookContext{Resource: m.cfg.Name, Operation: OpCreate, Stage: HookPost, Record: created, Scope: scope}); err != nil {
		return nil, err
	}
	return created, nil
}

func (m *Manager) Update(ctx context.Context, key string, record Record, scope map[string]interface{}) (Record, error) {
	if err := runHooks(ctx, m.cfg.Hooks.PreUpdate, HookContext{Resource: m.cfg.Name, Operation: OpUpdate, Stage: HookPre, PrimaryKey: key, Record: record, Scope: scope}); err != nil {
		return nil, err
	}
	updated, err := m.backend.Update(ctx, m.cfg, key, record, scope)
	if err != nil {
		return nil, err
	}
	if err := runHooks(ctx, m.cfg.Hooks.PostUpdate, HookContext{Resource: m.cfg.Name, Operation: OpUpdate, Stage: HookPost, PrimaryKey: key, Record: updated, Scope: scope}); err != nil {
		return nil, err
	}
	return updated, nil
}

func (m *Manager) Delete(ctx context.Context, key string, scope map[string]interface{}) error {
	if err := runHooks(ctx, m.cfg.Hooks.PreDelete, HookContext{Resource: m.cfg.Name, Operation: OpDelete, Stage: HookPre, PrimaryKey: key, Scope: scope}); err != nil {
		return err
	}
	if err := m.backend.Delete(ctx, m.cfg, key, scope); err != nil {
		return err
	}
	return runHooks(ctx, m.cfg.Hooks.PostDelete, HookContext{Resource: m.cfg.Name, Operation: OpDelete, Stage: HookPost, PrimaryKey: key, Scope: scope})
}

func (m *Manager) Bulk(ctx context.Context, action string, keys []string, scope map[string]interface{}) error {
	if err := runHooks(ctx, m.cfg.Hooks.PreBulk, HookContext{Resource: m.cfg.Name, Operation: OpBulk, Stage: HookPre, Selection: keys, Scope: scope}); err != nil {
		return err
	}
	if err := m.backend.Bulk(ctx, m.cfg, action, keys, scope); err != nil {
		return err
	}
	return runHooks(ctx, m.cfg.Hooks.PostBulk, HookContext{Resource: m.cfg.Name, Operation: OpBulk, Stage: HookPost, Selection: keys, Scope: scope})
}

func (m *Manager) ResolveRowActions(record Record) ([]ResolvedAction, error) {
	actions := append([]Action(nil), m.cfg.RowActions...)
	groups := actionGroupMap(m.cfg.ActionGroups)
	sort.SliceStable(actions, func(i, j int) bool {
		leftGroupOrder := groups[actions[i].Group].Order
		rightGroupOrder := groups[actions[j].Group].Order
		if leftGroupOrder != rightGroupOrder {
			if leftGroupOrder == 0 {
				return false
			}
			if rightGroupOrder == 0 {
				return true
			}
			return leftGroupOrder < rightGroupOrder
		}
		left := actions[i].Order
		right := actions[j].Order
		if left == right {
			return false
		}
		if left == 0 {
			return false
		}
		if right == 0 {
			return true
		}
		return left < right
	})
	out := make([]ResolvedAction, 0, len(actions))
	for _, action := range actions {
		if !actionVisible(action, record) {
			continue
		}
		url, err := renderURL(action.URLTemplate, record)
		if err != nil {
			return nil, err
		}
		out = append(out, ResolvedAction{
			Name:            action.Name,
			Label:           action.Label,
			Kind:            action.Kind,
			Method:          action.Method,
			URL:             url,
			Target:          action.Target,
			Icon:            action.Icon,
			Group:           action.Group,
			GroupLabel:      actionGroupLabel(action.Group, groups),
			Separator:       action.Separator,
			SeparatorBefore: action.SeparatorBefore,
			SeparatorAfter:  action.SeparatorAfter,
			ConfirmText:     action.ConfirmText,
		})
	}
	return out, nil
}

func actionGroupMap(groups []ActionGroup) map[string]ActionGroup {
	out := make(map[string]ActionGroup, len(groups))
	for _, group := range groups {
		name := strings.TrimSpace(group.Name)
		if name == "" {
			continue
		}
		out[name] = group
	}
	return out
}

func actionGroupLabel(name string, groups map[string]ActionGroup) string {
	groupName := strings.TrimSpace(name)
	if groupName == "" {
		return ""
	}
	group, ok := groups[groupName]
	if !ok || strings.TrimSpace(group.Label) == "" {
		return groupName
	}
	return group.Label
}

func actionVisible(action Action, record Record) bool {
	if action.Visible != nil {
		return action.Visible(record)
	}
	condition := strings.TrimSpace(action.VisibleWhen)
	if condition == "" {
		return true
	}
	if strings.EqualFold(condition, "false") || condition == "0" {
		return false
	}
	if strings.EqualFold(condition, "true") || condition == "1" {
		return true
	}
	for _, operator := range []string{"!=", "=", ":"} {
		if !strings.Contains(condition, operator) {
			continue
		}
		parts := strings.SplitN(condition, operator, 2)
		field := strings.TrimSpace(parts[0])
		expected := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		actual := fmt.Sprint(record[field])
		if operator == "!=" {
			return actual != expected
		}
		return actual == expected
	}
	return true
}

type ResolvedAction struct {
	Name            string `json:"name"`
	Label           string `json:"label"`
	Kind            string `json:"kind"`
	Method          string `json:"method"`
	URL             string `json:"url"`
	Target          string `json:"target"`
	Icon            string `json:"icon"`
	Group           string `json:"group"`
	GroupLabel      string `json:"group_label"`
	Separator       bool   `json:"separator"`
	SeparatorBefore bool   `json:"separator_before"`
	SeparatorAfter  bool   `json:"separator_after"`
	ConfirmText     string `json:"confirm_text"`
}

type ResolvedIndexColumn struct {
	Field          string `json:"field"`
	Label          string `json:"label"`
	LabelKey       string `json:"label_key"`
	Sortable       bool   `json:"sortable"`
	SortExpression string `json:"sort_expression"`
}

type ResolvedIndexCell struct {
	Field string      `json:"field"`
	Value string      `json:"value"`
	Raw   interface{} `json:"raw"`
}

type ResolvedIndexRow struct {
	Key     string              `json:"key"`
	Record  Record              `json:"record"`
	Cells   []ResolvedIndexCell `json:"cells"`
	Actions []ResolvedAction    `json:"actions"`
}

type IndexModel struct {
	Resource          string                `json:"resource"`
	TitleKey          string                `json:"title_key"`
	View              ViewMode              `json:"view"`
	AllowedViews      []ViewMode            `json:"allowed_views"`
	Fields            []Field               `json:"fields"`
	Columns           []ResolvedIndexColumn `json:"columns"`
	Rows              []ResolvedIndexRow    `json:"rows"`
	Result            ListResult            `json:"result"`
	BulkActions       []BulkAction          `json:"bulk_actions"`
	Breadcrumbs       []BreadcrumbItem      `json:"breadcrumbs"`
	SearchPlaceholder string                `json:"search_placeholder"`
	Pagination        PaginationConfig      `json:"pagination"`
	FilterFields      []Field               `json:"filter_fields"`
}

type FormModel struct {
	Resource    string
	Mode        string
	TitleKey    string
	Fields      []Field
	Record      Record
	Buttons     []FormButton
	Breadcrumbs []BreadcrumbItem
}

func runHooks(ctx context.Context, hooks []Hook, hookCtx HookContext) error {
	for _, hook := range hooks {
		if err := hook(ctx, hookCtx); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) filterFields() []Field {
	fields := make([]Field, 0, len(m.cfg.Fields))
	for _, field := range m.cfg.Fields {
		if field.Filterable {
			fields = append(fields, field)
		}
	}
	return fields
}

func (m *Manager) resolveColumns() []ResolvedIndexColumn {
	if len(m.cfg.Index.Columns) > 0 {
		columns := make([]ResolvedIndexColumn, 0, len(m.cfg.Index.Columns))
		for _, column := range m.cfg.Index.Columns {
			columns = append(columns, ResolvedIndexColumn{
				Field:          column.Field,
				Label:          column.Label,
				LabelKey:       column.LabelKey,
				Sortable:       column.Sortable,
				SortExpression: column.SortExpression,
			})
		}
		return columns
	}

	columns := make([]ResolvedIndexColumn, 0, len(m.cfg.Fields))
	for _, field := range m.cfg.Fields {
		if !field.Visible {
			continue
		}
		columns = append(columns, ResolvedIndexColumn{
			Field:          field.Name,
			Label:          field.Label,
			LabelKey:       field.LabelKey,
			Sortable:       field.Sortable,
			SortExpression: field.Name,
		})
	}
	return columns
}

func (m *Manager) resolveRows(records []Record, actions map[string][]ResolvedAction) []ResolvedIndexRow {
	columns := m.cfg.Index.Columns
	if len(columns) == 0 {
		for _, field := range m.cfg.Fields {
			if !field.Visible {
				continue
			}
			columns = append(columns, IndexColumn{
				Field:          field.Name,
				Label:          field.Label,
				LabelKey:       field.LabelKey,
				Sortable:       field.Sortable,
				SortExpression: field.Name,
			})
		}
	}

	rows := make([]ResolvedIndexRow, 0, len(records))
	for _, record := range records {
		key := fmt.Sprint(record[m.cfg.PrimaryKey])
		row := ResolvedIndexRow{
			Key:     key,
			Record:  record,
			Cells:   make([]ResolvedIndexCell, 0, len(columns)),
			Actions: actions[key],
		}
		for _, column := range columns {
			row.Cells = append(row.Cells, ResolvedIndexCell{
				Field: column.Field,
				Value: formatCellValue(column, record),
				Raw:   record[column.Field],
			})
		}
		rows = append(rows, row)
	}
	return rows
}

func formatCellValue(column IndexColumn, record Record) string {
	if column.DisplayFormatter != nil {
		return column.DisplayFormatter(record)
	}
	if strings.TrimSpace(column.DisplayTemplate) != "" {
		return renderDisplayTemplate(column.DisplayTemplate, column.Field, record)
	}
	if column.Formatter != nil {
		return column.Formatter(record)
	}

	value := record[column.Field]
	switch current := value.(type) {
	case nil:
		return ""
	case string:
		return current
	case []string:
		return strings.Join(current, ", ")
	case []interface{}:
		out := make([]string, 0, len(current))
		for _, item := range current {
			out = append(out, fmt.Sprint(item))
		}
		return strings.Join(out, ", ")
	default:
		return fmt.Sprint(current)
	}
}

func renderDisplayTemplate(tpl string, field string, record Record) string {
	value := fmt.Sprint(record[field])
	out := strings.ReplaceAll(tpl, "<row_value>", value)
	out = strings.ReplaceAll(out, "{row_value}", value)
	for key, current := range record {
		text := fmt.Sprint(current)
		out = strings.ReplaceAll(out, "<"+key+">", text)
		out = strings.ReplaceAll(out, "{"+key+"}", text)
	}
	return out
}

func renderURL(tpl string, record Record) (string, error) {
	out := tpl
	for key, value := range record {
		placeholder := "<" + key + ">"
		if strings.Contains(out, placeholder) {
			out = strings.ReplaceAll(out, placeholder, url.PathEscape(fmt.Sprint(value)))
		}
	}
	return out, nil
}

func parseSort(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseColumnFilters(values url.Values) []ColumnFilter {
	var out []ColumnFilter
	for key, value := range values {
		if !strings.HasPrefix(key, "cf_") || len(value) == 0 {
			continue
		}
		field := strings.TrimPrefix(key, "cf_")
		operator := FilterOperator(strings.TrimSpace(values.Get("op_" + field)))
		if operator == "" {
			operator = FilterOperatorLike
		}
		out = append(out, ColumnFilter{
			Field:    field,
			Operator: operator,
			Value:    strings.TrimSpace(value[0]),
		})
	}
	return out
}

func intQuery(values url.Values, key string, fallback int) int {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func normalizeFormButtons(buttons []FormButton) []FormButton {
	if len(buttons) > 0 {
		return buttons
	}
	return []FormButton{
		{Name: "save", LabelKey: "crud.action.save", Style: "primary"},
		{Name: "save_exit", LabelKey: "crud.action.save_and_exit", Redirect: "index", Style: "primary"},
		{Name: "save_new", LabelKey: "crud.action.save_and_new", Redirect: "new", Style: "outline-primary"},
		{Name: "cancel", LabelKey: "crud.action.cancel", Redirect: "index", Style: "outline-secondary"},
	}
}

func formBreadcrumbs(cfg Config, mode string, record Record) []BreadcrumbItem {
	items := []BreadcrumbItem{
		{LabelKey: "gmcore.dashboard", URL: "/"},
		{LabelKey: cfg.Index.TitleKey, URL: ""},
	}
	titleKey := cfg.Form.TitleCreateKey
	if mode == "edit" {
		titleKey = cfg.Form.TitleEditKey
	}
	items = append(items, BreadcrumbItem{LabelKey: titleKey})
	_ = record
	return items
}
