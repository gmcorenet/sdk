package gmcore_orm

// Package gmcore_orm provides a complete ORM wrapper around GORM with Repository pattern,
// Unit of Work, and Identity Map for consistent database operations across the application.
//
// Example usage:
//
//	// Define an entity
//	type User struct {
//	    ID        uint
//	    Name      string
//	    Email     string
//	}
//	func (u *User) TableName() string { return "users" }
//	func (u *User) GetID() interface{} { return u.ID }
//
//	// Setup
//	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
//	orm := New(db)
//	orm.AutoMigrate(&User{})
//
//	// Using Repository
//	repo := orm.Repository(&User{})
//	repo.Create(ctx, &User{Name: "John", Email: "john@example.com"})
//	user, _ := repo.Find(ctx, 1)
//
//	// Using Unit of Work for transactional operations
//	uow := orm.UnitOfWork()
//	uow.RegisterNew(&User{Name: "Jane", Email: "jane@example.com"})
//	uow.RegisterDirty(user) // Modified
//	uow.Commit(ctx)
//
//	// Using Query Builder
//	var users []User
//	orm.Query(&User{}).Where("name LIKE ?", "%John%").Find(ctx, &users)

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrRecordNotFound = gorm.ErrRecordNotFound
	ErrInvalidEntity  = errors.New("entity must implement Entity interface")
	ErrNoTransaction  = errors.New("no active transaction")
)

// Entity interface for all database entities.
// Entities must be registered with the ORM to enable repository operations.
type Entity interface {
	// TableName returns the database table name for this entity.
	TableName() string
	// GetID returns the primary key value of the entity.
	GetID() interface{}
}

// Repository provides CRUD operations for a specific entity type.
type Repository interface {
	// Find retrieves an entity by its primary key.
	Find(ctx context.Context, id interface{}) (Entity, error)
	// FindAll retrieves all entities.
	FindAll(ctx context.Context) ([]Entity, error)
	// Create inserts a new entity.
	Create(ctx context.Context, entity Entity) error
	// Update modifies an existing entity.
	Update(ctx context.Context, entity Entity) error
	// Delete removes an entity.
	Delete(ctx context.Context, entity Entity) error
	// Save creates or updates an entity based on whether it has an ID.
	Save(ctx context.Context, entity Entity) error
}

// QueryExecutor defines the interface for query building.
type QueryExecutor interface {
	// Where adds a WHERE condition.
	Where(condition string, args ...interface{}) QueryExecutor
	// OrderBy adds ORDER BY clause.
	OrderBy(field string) QueryExecutor
	// Limit sets the maximum number of records.
	Limit(limit int) QueryExecutor
	// Offset sets the number of records to skip.
	Offset(offset int) QueryExecutor
	// Find executes the query and populates the destination.
	Find(ctx context.Context, dest interface{}) error
	// First retrieves the first matching record.
	First(ctx context.Context, dest interface{}) error
	// Count returns the number of matching records.
	Count(ctx context.Context) (int64, error)
}

// DB wraps a GORM database connection with additional ORM features.
type DB struct {
	db *gorm.DB
}

// New creates a new ORM wrapper around a GORM database.
func New(db *gorm.DB) *DB {
	return &DB{db: db}
}

// DB returns the underlying GORM database connection.
func (o *DB) DB() *gorm.DB {
	return o.db
}

// Close closes the underlying database connection.
func (o *DB) Close() error {
	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Transaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// If the function succeeds, the transaction is committed.
func (o *DB) Transaction(fn func(*gorm.DB) error) error {
	return o.db.Transaction(fn)
}

// TransactionWithORMCallback executes a function within a database transaction
// and passes an ORM wrapper to the callback.
func (o *DB) TransactionWithORMCallback(fn func(*DB) error) error {
	return o.db.Transaction(func(tx *gorm.DB) error {
		return fn(&DB{db: tx})
	})
}

// WithTransaction creates a new DB instance with an active transaction.
func (o *DB) WithTransaction(tx *gorm.DB) *DB {
	return &DB{db: tx}
}

// AutoMigrate runs auto migration for the given entities.
func (o *DB) AutoMigrate(entities ...Entity) error {
	if len(entities) == 0 {
		return nil
	}
	models := make([]interface{}, len(entities))
	for i, e := range entities {
		models[i] = e
	}
	return o.db.AutoMigrate(models...)
}

// Repository returns a Repository for the given entity type.
func (o *DB) Repository(entity Entity) Repository {
	return newRepository(o.db, entity)
}

// Query returns a QueryBuilder for the given entity type.
func (o *DB) Query(entity Entity) QueryBuilder {
	return QueryBuilder{
		db:        o.db,
		tableName: entity.TableName(),
	}
}

// UnitOfWork returns a new UnitOfWork instance.
func (o *DB) UnitOfWork() *UnitOfWork {
	return newUnitOfWork(o.db)
}

// IdentityMap returns a new IdentityMap instance.
func (o *DB) IdentityMap() *IdentityMap {
	return newIdentityMap()
}

// GORMRepository implements Repository using GORM.
type GORMRepository struct {
	db         *gorm.DB
	entityType reflect.Type
	tableName  string
}

func newRepository(db *gorm.DB, entity Entity) *GORMRepository {
	if entity == nil {
		panic(ErrInvalidEntity)
	}
	return &GORMRepository{
		db:         db,
		entityType: reflect.TypeOf(entity).Elem(),
		tableName:  entity.TableName(),
	}
}

// Find implements Repository.
func (r *GORMRepository) Find(ctx context.Context, id interface{}) (Entity, error) {
	entity := reflect.New(r.entityType).Interface().(Entity)
	result := r.db.WithContext(ctx).Where("id = ?", id).First(entity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("record with id %v not found", id)
		}
		return nil, result.Error
	}
	return entity, nil
}

// FindAll implements Repository.
func (r *GORMRepository) FindAll(ctx context.Context) ([]Entity, error) {
	var entities []Entity
	result := r.db.WithContext(ctx).Find(&entities)
	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// Create implements Repository.
func (r *GORMRepository) Create(ctx context.Context, entity Entity) error {
	if entity == nil {
		return errors.New("entity cannot be nil")
	}
	result := r.db.WithContext(ctx).Create(entity)
	return result.Error
}

// Update implements Repository.
func (r *GORMRepository) Update(ctx context.Context, entity Entity) error {
	if entity == nil {
		return errors.New("entity cannot be nil")
	}
	result := r.db.WithContext(ctx).Save(entity)
	return result.Error
}

// Delete implements Repository.
func (r *GORMRepository) Delete(ctx context.Context, entity Entity) error {
	if entity == nil {
		return errors.New("entity cannot be nil")
	}
	result := r.db.WithContext(ctx).Delete(entity)
	return result.Error
}

// Save implements Repository.
func (r *GORMRepository) Save(ctx context.Context, entity Entity) error {
	if entity == nil {
		return errors.New("entity cannot be nil")
	}
	id := entity.GetID()
	if id == nil || id == 0 {
		return r.Create(ctx, entity)
	}
	return r.Update(ctx, entity)
}

// FindBy executes a query with custom WHERE conditions.
func (r *GORMRepository) FindBy(ctx context.Context, where map[string]interface{}) ([]Entity, error) {
	query := r.db.WithContext(ctx)
	for field, value := range where {
		query = query.Where(field+" = ?", value)
	}
	var entities []Entity
	result := query.Find(&entities)
	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// QueryBuilder provides a chainable query building interface.
type QueryBuilder struct {
	db         *gorm.DB
	tableName  string
	conditions []string
	args       []interface{}
	orderBy    []string
	limit      int
	offset     int
}

// Where adds a WHERE condition.
func (qb QueryBuilder) Where(condition string, args ...interface{}) QueryBuilder {
	qb.conditions = append(qb.conditions, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy adds an ORDER BY clause.
func (qb QueryBuilder) OrderBy(field string) QueryBuilder {
	qb.orderBy = append(qb.orderBy, field)
	return qb
}

// Limit sets the maximum number of records.
func (qb QueryBuilder) Limit(limit int) QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the number of records to skip.
func (qb QueryBuilder) Offset(offset int) QueryBuilder {
	qb.offset = offset
	return qb
}

// Find executes the query and populates dest with results.
func (qb QueryBuilder) Find(ctx context.Context, dest interface{}) error {
	query := qb.db.WithContext(ctx).Table(qb.tableName)

	if len(qb.conditions) > 0 {
		query = query.Where(strings.Join(qb.conditions, " AND "), qb.args...)
	}

	if len(qb.orderBy) > 0 {
		query = query.Order(strings.Join(qb.orderBy, ", "))
	}

	if qb.limit > 0 {
		query = query.Limit(qb.limit)
	}

	if qb.offset > 0 {
		query = query.Offset(qb.offset)
	}

	return query.Find(dest).Error
}

// First retrieves the first matching record.
func (qb QueryBuilder) First(ctx context.Context, dest interface{}) error {
	query := qb.db.WithContext(ctx).Table(qb.tableName)

	if len(qb.conditions) > 0 {
		query = query.Where(strings.Join(qb.conditions, " AND "), qb.args...)
	}

	if len(qb.orderBy) > 0 {
		query = query.Order(strings.Join(qb.orderBy, ", "))
	}

	return query.First(dest).Error
}

// Count returns the number of matching records.
func (qb QueryBuilder) Count(ctx context.Context) (int64, error) {
	var count int64
	query := qb.db.WithContext(ctx).Table(qb.tableName)

	if len(qb.conditions) > 0 {
		query = query.Where(strings.Join(qb.conditions, " AND "), qb.args...)
	}

	err := query.Count(&count).Error
	return count, err
}

// Create inserts a new record.
func (qb QueryBuilder) Create(ctx context.Context, record interface{}) error {
	return qb.db.WithContext(ctx).Table(qb.tableName).Create(record).Error
}

// Delete removes all matching records.
func (qb QueryBuilder) Delete(ctx context.Context) error {
	query := qb.db.WithContext(ctx).Table(qb.tableName)

	if len(qb.conditions) > 0 {
		query = query.Where(strings.Join(qb.conditions, " AND "), qb.args...)
	}

	return query.Delete(nil).Error
}

// Update updates all matching records with the given values.
func (qb QueryBuilder) Update(ctx context.Context, values map[string]interface{}) error {
	query := qb.db.WithContext(ctx).Table(qb.tableName)

	if len(qb.conditions) > 0 {
		query = query.Where(strings.Join(qb.conditions, " AND "), qb.args...)
	}

	return query.UpdateColumns(values).Error
}

// RawQuery executes a raw query and populates dest with results.
func (qb QueryBuilder) RawQuery(query string, args ...interface{}) error {
	return qb.db.Raw(query, args...).Find(qb.tableName).Error
}

// UnitOfWork implements the Unit of Work pattern for transactional operations.
type UnitOfWork struct {
	db     *gorm.DB
	clean  map[string]Entity
	dirty  []Entity
	added  []Entity
	removed []Entity
}

func newUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{
		db:    db,
		clean: make(map[string]Entity),
	}
}

// RegisterClean registers an entity that was read from the database.
func (u *UnitOfWork) RegisterClean(entity Entity) {
	if entity == nil {
		return
	}
	key := u.getKey(entity)
	u.clean[key] = entity
}

// RegisterDirty registers an entity that has been modified.
func (u *UnitOfWork) RegisterDirty(entity Entity) {
	if entity == nil {
		return
	}
	u.dirty = append(u.dirty, entity)
}

// RegisterNew registers an entity that is new and should be inserted.
func (u *UnitOfWork) RegisterNew(entity Entity) {
	if entity == nil {
		return
	}
	u.added = append(u.added, entity)
}

// RegisterRemoved registers an entity that should be deleted.
func (u *UnitOfWork) RegisterRemoved(entity Entity) {
	if entity == nil {
		return
	}
	u.removed = append(u.removed, entity)
}

// Commit executes all registered operations in a single transaction.
func (u *UnitOfWork) Commit(ctx context.Context) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, e := range u.added {
			if err := tx.Create(e).Error; err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
		}

		for _, e := range u.dirty {
			if err := tx.Save(e).Error; err != nil {
				return fmt.Errorf("failed to update: %w", err)
			}
		}

		for _, e := range u.removed {
			if err := tx.Delete(e).Error; err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
		}

		return nil
	})
}

// Clear resets the UnitOfWork without committing.
func (u *UnitOfWork) Clear() {
	u.added = nil
	u.dirty = nil
	u.removed = nil
}

func (u *UnitOfWork) getKey(e Entity) string {
	return fmt.Sprintf("%s:%v", e.TableName(), e.GetID())
}

// IdentityMap tracks entities by ID to avoid duplicate loading.
type IdentityMap struct {
	entities map[string]Entity
}

func newIdentityMap() *IdentityMap {
	return &IdentityMap{
		entities: make(map[string]Entity),
	}
}

// Put adds or updates an entity in the identity map.
func (m *IdentityMap) Put(entity Entity) {
	if entity == nil {
		return
	}
	key := fmt.Sprintf("%s:%v", entity.TableName(), entity.GetID())
	m.entities[key] = entity
}

// Get retrieves an entity by type and ID.
func (m *IdentityMap) Get(tableName string, id interface{}) (Entity, bool) {
	key := fmt.Sprintf("%s:%v", tableName, id)
	e, ok := m.entities[key]
	return e, ok
}

// Remove removes an entity from the identity map.
func (m *IdentityMap) Remove(tableName string, id interface{}) {
	key := fmt.Sprintf("%s:%v", tableName, id)
	delete(m.entities, key)
}

// Clear removes all entities from the identity map.
func (m *IdentityMap) Clear() {
	m.entities = make(map[string]Entity)
}

// Size returns the number of entities in the identity map.
func (m *IdentityMap) Size() int {
	return len(m.entities)
}

// RepositoryRegistry manages repositories for different entity types.
type RepositoryRegistry struct {
	db      *gorm.DB
	repos   map[string]Repository
	entities map[string]reflect.Type
}

// NewRegistry creates a new repository registry.
func NewRegistry(db *gorm.DB) *RepositoryRegistry {
	return &RepositoryRegistry{
		db:       db,
		repos:    make(map[string]Repository),
		entities: make(map[string]reflect.Type),
	}
}

// Register registers an entity type for repository access.
func (r *RepositoryRegistry) Register(entity Entity) {
	name := reflect.TypeOf(entity).Elem().Name()
	r.entities[name] = reflect.TypeOf(entity).Elem()
}

// GetRepository returns the repository for an entity type.
func (r *RepositoryRegistry) GetRepository(name string) Repository {
	if repo, ok := r.repos[name]; ok {
		return repo
	}
	return nil
}

// AddRepository manually adds a repository to the registry.
func (r *RepositoryRegistry) AddRepository(name string, repo Repository) {
	r.repos[name] = repo
}

// CreateRepository creates and registers a repository for an entity.
func (r *RepositoryRegistry) CreateRepository(entity Entity) Repository {
	name := reflect.TypeOf(entity).Elem().Name()
	repo := newRepository(r.db, entity)
	r.repos[name] = repo
	return repo
}

// AutoMigrate runs auto migration for all registered entities.
func (r *RepositoryRegistry) AutoMigrate() error {
	for _, entityType := range r.entities {
		entity := reflect.New(entityType).Interface().(Entity)
		if err := r.db.AutoMigrate(entity); err != nil {
			return err
		}
	}
	return nil
}
