package gmcore_doctrine

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type Entity interface {
	TableName() string
}

type Repository interface {
	Find(id interface{}) (Entity, error)
	FindAll() ([]Entity, error)
	Save(Entity) error
	Delete(Entity) error
}

type EntityManager struct {
	entities map[string]Entity
	repos    map[string]Repository
}

func NewEntityManager() *EntityManager {
	return &EntityManager{
		entities: make(map[string]Entity),
		repos:    make(map[string]Repository),
	}
}

func (em *EntityManager) Register(entity Entity) {
	name := reflect.TypeOf(entity).Elem().Name()
	em.entities[name] = entity
}

func (em *EntityManager) GetRepository(name string) Repository {
	if repo, ok := em.repos[name]; ok {
		return repo
	}
	return nil
}

func (em *EntityManager) AddRepository(name string, repo Repository) {
	em.repos[name] = repo
}

type QueryBuilder struct {
	entityName   string
	conditions   []string
	orderBy      []string
	limit        int
	offset       int
	params       []interface{}
	useParameter bool
}

var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func sanitizeIdentifier(identifier string) string {
	if !identifierRegex.MatchString(identifier) {
		return ""
	}
	return identifier
}

func NewQueryBuilder(entity string) *QueryBuilder {
	return &QueryBuilder{
		entityName:   entity,
		params:       make([]interface{}, 0),
		useParameter: true,
	}
}

func (qb *QueryBuilder) Where(condition string) *QueryBuilder {
	qb.conditions = append(qb.conditions, condition)
	return qb
}

func (qb *QueryBuilder) WhereParams(condition string, params ...interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, condition)
	qb.params = append(qb.params, params...)
	return qb
}

func (qb *QueryBuilder) OrderBy(field string) *QueryBuilder {
	qb.orderBy = append(qb.orderBy, field)
	return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

func (qb *QueryBuilder) Build() (string, []interface{}) {
	var query strings.Builder
	query.WriteString("SELECT * FROM ")
	query.WriteString(sanitizeIdentifier(qb.entityName))

	if len(qb.conditions) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(qb.conditions[0])
	}
	if len(qb.orderBy) > 0 {
		query.WriteString(" ORDER BY ")
		orderClause := qb.orderBy[0]
		if !strings.Contains(orderClause, " ") && !strings.Contains(orderClause, "(") {
			orderClause = sanitizeIdentifier(orderClause)
		}
		query.WriteString(orderClause)
	}
	if qb.limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", qb.limit))
	}
	if qb.offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", qb.offset))
	}
	return query.String(), qb.params
}

type Table struct {
	Name    string
	Columns []Column
}

type Column struct {
	Name    string
	Type    string
	Primary bool
	NotNull bool
	Default interface{}
}

func NewTable(name string) *Table {
	return &Table{Name: name, Columns: make([]Column, 0)}
}

func (t *Table) AddColumn(name, colType string) *Column {
	t.Columns = append(t.Columns, Column{Name: name, Type: colType})
	return &t.Columns[len(t.Columns)-1]
}

func (c *Column) SetPrimary() *Column  { c.Primary = true; return c }
func (c *Column) SetNotNull() *Column { c.NotNull = true; return c }
func (c *Column) SetDefault(v interface{}) *Column { c.Default = v; return c }

type Schema struct {
	tables map[string]*Table
}

func NewSchema() *Schema {
	return &Schema{tables: make(map[string]*Table)}
}

func (s *Schema) AddTable(table *Table) {
	s.tables[table.Name] = table
}

func (s *Schema) GetTable(name string) *Table {
	return s.tables[name]
}

type UnitOfWork struct {
	entityManager *EntityManager
	clean         map[string]Entity
	dirty         []Entity
	removed       []Entity
	added         []Entity
}

func NewUnitOfWork(em *EntityManager) *UnitOfWork {
	return &UnitOfWork{
		entityManager: em,
		clean:         make(map[string]Entity),
	}
}

func (u *UnitOfWork) RegisterClean(entity Entity) {
	key := getTableName(entity)
	u.clean[key] = entity
}

func (u *UnitOfWork) RegisterDirty(entity Entity) {
	u.dirty = append(u.dirty, entity)
}

func (u *UnitOfWork) RegisterRemoved(entity Entity) {
	u.removed = append(u.removed, entity)
}

func (u *UnitOfWork) RegisterNew(entity Entity) {
	u.added = append(u.added, entity)
}

func (u *UnitOfWork) Commit() error {
	for _, e := range u.added {
		if repo := u.entityManager.GetRepository(getTableName(e)); repo != nil {
			repo.Save(e)
		}
	}
	for _, e := range u.dirty {
		if repo := u.entityManager.GetRepository(getTableName(e)); repo != nil {
			repo.Save(e)
		}
	}
	for _, e := range u.removed {
		if repo := u.entityManager.GetRepository(getTableName(e)); repo != nil {
			repo.Delete(e)
		}
	}
	u.clear()
	return nil
}

func (u *UnitOfWork) clear() {
	u.added = nil
	u.dirty = nil
	u.removed = nil
}

func getTableName(e Entity) string {
	return e.TableName()
}

type IdentityMap struct {
	entities map[string]Entity
}

func NewIdentityMap() *IdentityMap {
	return &IdentityMap{entities: make(map[string]Entity)}
}

func (m *IdentityMap) Add(id string, entity Entity) {
	m.entities[id] = entity
}

func (m *IdentityMap) Get(id string) (Entity, bool) {
	e, ok := m.entities[id]
	return e, ok
}

func (m *IdentityMap) Remove(id string) {
	delete(m.entities, id)
}
