package gmcore_crud

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gmcore_form "github.com/gmcorenet/sdk/gmcore-form"
)

type stubBackend struct{}

func (stubBackend) Kind() BackendKind { return BackendArray }
func (stubBackend) List(context.Context, Config, ListParams) ([]Record, error) {
	return nil, nil
}
func (stubBackend) Get(context.Context, Config, string, map[string]interface{}) (Record, error) {
	return nil, nil
}
func (stubBackend) Create(context.Context, Config, Record, map[string]interface{}) (Record, error) {
	return nil, nil
}
func (stubBackend) Update(context.Context, Config, string, Record, map[string]interface{}) (Record, error) {
	return nil, nil
}
func (stubBackend) Delete(context.Context, Config, string, map[string]interface{}) error {
	return nil
}
func (stubBackend) Bulk(context.Context, Config, string, []string, map[string]interface{}) error {
	return nil
}

func TestNewRequiresPrimaryKeyNamedID(t *testing.T) {
	_, err := New(Config{
		Name:           "notes",
		Backend:        BackendArray,
		PrimaryKey:     "uuid",
		PrimaryKeyType: PrimaryKeyUUID,
		Fields:         []Field{{Name: "uuid", Type: "uuid"}},
	}, stubBackend{})
	if err == nil || !strings.Contains(err.Error(), "must be named id") {
		t.Fatalf("expected primary key name validation error, got %v", err)
	}
}

func TestNewRequiresPrimaryKeyFieldDefinition(t *testing.T) {
	_, err := New(Config{
		Name:           "notes",
		Backend:        BackendArray,
		PrimaryKey:     "id",
		PrimaryKeyType: PrimaryKeyUUID,
		Fields:         []Field{{Name: "title", Type: "string"}},
	}, stubBackend{})
	if err == nil || !strings.Contains(err.Error(), "must define the primary key field id") {
		t.Fatalf("expected primary key field validation error, got %v", err)
	}
}

func TestNewRequiresUUIDOrIntPrimaryKeyType(t *testing.T) {
	_, err := New(Config{
		Name:           "notes",
		Backend:        BackendArray,
		PrimaryKey:     "id",
		PrimaryKeyType: "string",
		Fields:         []Field{{Name: "id", Type: "string"}},
	}, stubBackend{})
	if err == nil || !strings.Contains(err.Error(), "must be uuid or int") {
		t.Fatalf("expected primary key type validation error, got %v", err)
	}
}

func TestNewAllowsUUIDOrIntIDPrimaryKey(t *testing.T) {
	for _, pkType := range []PrimaryKeyType{PrimaryKeyUUID, PrimaryKeyInt} {
		manager, err := New(Config{
			Name:           "notes",
			Backend:        BackendArray,
			PrimaryKey:     "id",
			PrimaryKeyType: pkType,
			Fields:         []Field{{Name: "id", Type: "string"}, {Name: "title", Type: "string"}},
		}, stubBackend{})
		if err != nil {
			t.Fatalf("expected %s primary key to be valid, got %v", pkType, err)
		}
		if manager == nil {
			t.Fatalf("expected manager for %s primary key", pkType)
		}
	}
}

func TestDefaultFilterOperatorForFieldUsesEqForIdentifierAndNumericLikeForText(t *testing.T) {
	cfg := Config{
		Fields: []Field{
			{Name: "id", Type: "uuid"},
			{Name: "priority", Type: "int"},
			{Name: "updated_at", Type: "datetime"},
			{Name: "title", Type: "string"},
		},
	}
	if got := defaultFilterOperatorForField(cfg, "id"); got != FilterOperatorEq {
		t.Fatalf("expected id filter to use eq, got %s", got)
	}
	if got := defaultFilterOperatorForField(cfg, "priority"); got != FilterOperatorEq {
		t.Fatalf("expected priority filter to use eq, got %s", got)
	}
	if got := defaultFilterOperatorForField(cfg, "updated_at"); got != FilterOperatorEq {
		t.Fatalf("expected updated_at filter to use eq, got %s", got)
	}
	if got := defaultFilterOperatorForField(cfg, "title"); got != FilterOperatorLike {
		t.Fatalf("expected title filter to use like, got %s", got)
	}
}

func TestParseListParamsRespectsExplicitSimpleFilterOperator(t *testing.T) {
	manager, err := New(Config{
		Name:           "notes",
		Backend:        BackendArray,
		PrimaryKey:     "id",
		PrimaryKeyType: PrimaryKeyUUID,
		Fields: []Field{
			{Name: "id", Type: "uuid", Filterable: true},
			{Name: "title", Type: "string", Filterable: true},
		},
	}, stubBackend{})
	if err != nil {
		t.Fatalf("unexpected error creating manager: %v", err)
	}
	values := url.Values{}
	values.Set("filter_id", "27")
	values.Set("filter_id_op", "eq")
	request := httptest.NewRequest("GET", "/notes?"+values.Encode(), nil)
	params := manager.ParseListParams(request)
	if _, ok := params.Filters["id"]; ok {
		t.Fatalf("expected simple filter to move into column filters when operator is explicit")
	}
	if len(params.ColumnFilters) != 1 {
		t.Fatalf("expected one column filter, got %d", len(params.ColumnFilters))
	}
	if params.ColumnFilters[0].Field != "id" || params.ColumnFilters[0].Operator != FilterOperatorEq || params.ColumnFilters[0].Value != "27" {
		t.Fatalf("unexpected column filter parsed: %+v", params.ColumnFilters[0])
	}
}

type queryNamedStubBackend struct {
	stubBackend
	query string
}

func (b queryNamedStubBackend) IndexQueryName() string {
	return b.query
}

func TestManagerDerivesQueryNameFromBackend(t *testing.T) {
	manager, err := New(Config{
		Name:           "notes",
		Backend:        BackendArray,
		PrimaryKey:     "id",
		PrimaryKeyType: PrimaryKeyUUID,
		Fields:         []Field{{Name: "id", Type: "uuid"}, {Name: "title", Type: "string"}},
	}, queryNamedStubBackend{query: "notes_index"})
	if err != nil {
		t.Fatalf("unexpected error creating manager: %v", err)
	}
	if got := manager.queryName(); got != "notes_index" {
		t.Fatalf("expected backend query name, got %q", got)
	}
}

func TestNewRequiresDisplayAndValueFieldsForRelations(t *testing.T) {
	_, err := New(Config{
		Name:           "notes",
		Backend:        BackendArray,
		PrimaryKey:     "id",
		PrimaryKeyType: PrimaryKeyUUID,
		Fields:         []Field{{Name: "id", Type: "uuid"}, {Name: "owner", Type: "string", Relation: "owner"}},
		Relations: []Relation{
			{Name: "owner", Type: RelationBelongsTo, LocalField: "owner", ValueField: "email"},
		},
	}, stubBackend{})
	if err == nil || !strings.Contains(err.Error(), "missing display field") {
		t.Fatalf("expected missing display field error, got %v", err)
	}
}

func TestManagerResolvesRelationByName(t *testing.T) {
	manager, err := New(Config{
		Name:           "notes",
		Backend:        BackendArray,
		PrimaryKey:     "id",
		PrimaryKeyType: PrimaryKeyUUID,
		Fields:         []Field{{Name: "id", Type: "uuid"}, {Name: "owner", Type: "string", Relation: "owner"}},
		Relations: []Relation{
			{Name: "owner", Type: RelationBelongsTo, LocalField: "owner", ValueField: "email", DisplayField: "display_name"},
		},
	}, stubBackend{})
	if err != nil {
		t.Fatalf("unexpected error creating manager: %v", err)
	}
	relation, ok := manager.Relation("owner")
	if !ok || relation.DisplayField != "display_name" {
		t.Fatalf("unexpected relation resolution: %#v %v", relation, ok)
	}
}

func TestEffectiveFormDefinitionUsesMultiselectForHasManyRelations(t *testing.T) {
	cfg := Config{
		Fields: []Field{
			{Name: "id", Type: "uuid"},
			{Name: "watchers", Type: "string", Relation: "watchers"},
		},
		Relations: []Relation{
			{Name: "watchers", Type: RelationHasMany, LocalField: "id", ForeignKey: "note_id", ValueField: "email", DisplayField: "display_name"},
		},
	}
	form := EffectiveFormDefinition("notes", "notes.title", gmcoreform.Definition{}, cfg)
	if len(form.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(form.Fields))
	}
	watchers := form.Fields[1]
	if watchers.Widget != "multiselect" {
		t.Fatalf("expected multiselect widget for has_many relation, got %q", watchers.Widget)
	}
	if !watchers.Multiple {
		t.Fatalf("expected has_many relation to be marked multiple")
	}
}

func TestFormatCellValueUsesDisplayFormatterFirst(t *testing.T) {
	record := Record{"status": "published"}
	column := IndexColumn{
		Field: "status",
		DisplayFormatter: func(_ Record) string {
			return "Published"
		},
		Formatter: func(_ Record) string {
			return "ignored"
		},
	}
	if got := formatCellValue(column, record); got != "Published" {
		t.Fatalf("expected display formatter value, got %q", got)
	}
}

func TestFormatCellValueUsesDisplayTemplate(t *testing.T) {
	record := Record{"amount": 27, "currency": "EUR", "first_name": "Ada", "last_name": "Lovelace"}
	if got := formatCellValue(IndexColumn{Field: "amount", DisplayTemplate: "<row_value>€"}, record); got != "27€" {
		t.Fatalf("expected row value display template, got %q", got)
	}
	if got := formatCellValue(IndexColumn{Field: "first_name", DisplayTemplate: "{first_name} {last_name}"}, record); got != "Ada Lovelace" {
		t.Fatalf("expected record display template, got %q", got)
	}
}

func TestResolveRowActionsSupportsCustomActionMetadataAndVisibility(t *testing.T) {
	manager, err := New(Config{
		Name:           "users",
		Backend:        BackendArray,
		PrimaryKey:     "id",
		PrimaryKeyType: PrimaryKeyUUID,
		Fields:         []Field{{Name: "id", Type: "uuid"}, {Name: "email", Type: "string"}, {Name: "status", Type: "string"}},
		ActionGroups:   []ActionGroup{{Name: "account", Label: "Account actions", Order: 1}},
		RowActions: []Action{
			{Name: "edit", Label: "Edit"},
			{Name: "hidden_callback", Label: "Hidden", Visible: func(Record) bool { return false }},
			{Name: "hidden_condition", Label: "Hidden", VisibleWhen: "status=disabled"},
			{Name: "impersonate", Label: "Impersonate", URLTemplate: "/impersonate/<id>", Target: "_self", Icon: "/assets/custom/impersonate.svg", Order: 1, Group: "account", SeparatorBefore: true, ConfirmText: "Impersonate user?", VisibleWhen: "status=active"},
			{Name: "website", Label: "Website", URLTemplate: "https://example.test/u/<email>", Target: "_blank", Order: 2, SeparatorAfter: true},
		},
	}, stubBackend{})
	if err != nil {
		t.Fatalf("unexpected manager error: %v", err)
	}
	actions, err := manager.ResolveRowActions(Record{"id": "user 1", "email": "root@example.test", "status": "active"})
	if err != nil {
		t.Fatalf("resolve actions: %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("expected 3 visible actions, got %#v", actions)
	}
	if actions[0].Name != "impersonate" || actions[1].Name != "website" || actions[2].Name != "edit" {
		t.Fatalf("unexpected action order: %#v", actions)
	}
	if actions[0].URL != "/impersonate/user%201" || actions[0].Target != "_self" || actions[0].Icon != "/assets/custom/impersonate.svg" || actions[0].ConfirmText != "Impersonate user?" {
		t.Fatalf("unexpected impersonate metadata: %#v", actions[0])
	}
	if actions[0].Group != "account" || actions[0].GroupLabel != "Account actions" || !actions[0].SeparatorBefore {
		t.Fatalf("unexpected action grouping metadata: %#v", actions[0])
	}
	if actions[1].URL != "https://example.test/u/root@example.test" || actions[1].Target != "_blank" {
		t.Fatalf("unexpected website metadata: %#v", actions[1])
	}
	if !actions[1].SeparatorAfter {
		t.Fatalf("expected separator after website action: %#v", actions[1])
	}
}
