package gmcore_orm

import (
	"testing"
)

type TestEntity struct {
	ID   uint
	Name string
}

func (e *TestEntity) TableName() string {
	return "test_entities"
}

func (e *TestEntity) GetID() interface{} {
	return e.ID
}

func TestNewORM(t *testing.T) {
	db := &DB{}
	if db == nil {
		t.Error("expected non-nil DB")
	}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry(nil)
	if registry == nil {
		t.Error("expected non-nil registry")
	}
	if registry.repos == nil {
		t.Error("expected repos map to be initialized")
	}
	if registry.entities == nil {
		t.Error("expected entities map to be initialized")
	}
}

func TestRepositoryRegistryRegister(t *testing.T) {
	registry := NewRegistry(nil)
	entity := &TestEntity{}
	registry.Register(entity)

	if len(registry.entities) != 1 {
		t.Errorf("expected 1 entity, got %d", len(registry.entities))
	}
}

func TestRepositoryRegistryGetRepository(t *testing.T) {
	registry := NewRegistry(nil)

	repo := registry.GetRepository("TestEntity")
	if repo != nil {
		t.Error("expected nil for non-existent repository")
	}
}

func TestGORMRepositoryCreation(t *testing.T) {
	entity := &TestEntity{}
	repo := newRepository(nil, entity)
	if repo == nil {
		t.Error("expected non-nil repository")
	}
	if repo.tableName != "test_entities" {
		t.Errorf("expected table name 'test_entities', got %s", repo.tableName)
	}
}

func TestUnitOfWorkCreation(t *testing.T) {
	uow := newUnitOfWork(nil)
	if uow == nil {
		t.Error("expected non-nil unit of work")
	}
	if uow.clean == nil {
		t.Error("expected clean map to be initialized")
	}
}

func TestUnitOfWorkRegisterClean(t *testing.T) {
	uow := newUnitOfWork(nil)
	entity := &TestEntity{ID: 1, Name: "Test"}
	uow.RegisterClean(entity)

	if len(uow.clean) != 1 {
		t.Errorf("expected 1 clean entity, got %d", len(uow.clean))
	}
}

func TestUnitOfWorkRegisterDirty(t *testing.T) {
	uow := newUnitOfWork(nil)
	entity := &TestEntity{ID: 1, Name: "Test"}
	uow.RegisterDirty(entity)

	if len(uow.dirty) != 1 {
		t.Errorf("expected 1 dirty entity, got %d", len(uow.dirty))
	}
}

func TestUnitOfWorkRegisterRemoved(t *testing.T) {
	uow := newUnitOfWork(nil)
	entity := &TestEntity{ID: 1, Name: "Test"}
	uow.RegisterRemoved(entity)

	if len(uow.removed) != 1 {
		t.Errorf("expected 1 removed entity, got %d", len(uow.removed))
	}
}

func TestUnitOfWorkRegisterNew(t *testing.T) {
	uow := newUnitOfWork(nil)
	entity := &TestEntity{ID: 0, Name: "Test"}
	uow.RegisterNew(entity)

	if len(uow.added) != 1 {
		t.Errorf("expected 1 added entity, got %d", len(uow.added))
	}
}

func TestIdentityMapCreation(t *testing.T) {
	im := newIdentityMap()
	if im == nil {
		t.Error("expected non-nil identity map")
	}
	if im.entities == nil {
		t.Error("expected entities map to be initialized")
	}
}

func TestIdentityMapPut(t *testing.T) {
	im := newIdentityMap()
	entity := &TestEntity{ID: 1, Name: "Test"}
	im.Put(entity)

	if len(im.entities) != 1 {
		t.Errorf("expected 1 entity, got %d", len(im.entities))
	}
}

func TestIdentityMapGet(t *testing.T) {
	im := newIdentityMap()
	entity := &TestEntity{ID: 1, Name: "Test"}
	im.Put(entity)

	retrieved, ok := im.Get("test_entities", 1)
	if !ok {
		t.Error("expected to find entity")
	}
	if retrieved.GetID() != entity.GetID() {
		t.Error("retrieved entity does not match")
	}
}

func TestIdentityMapRemove(t *testing.T) {
	im := newIdentityMap()
	entity := &TestEntity{ID: 1, Name: "Test"}
	im.Put(entity)
	im.Remove("test_entities", 1)

	if len(im.entities) != 0 {
		t.Errorf("expected 0 entities, got %d", len(im.entities))
	}
}

func TestIdentityMapClear(t *testing.T) {
	im := newIdentityMap()
	entity := &TestEntity{ID: 1, Name: "Test"}
	im.Put(entity)
	im.Clear()

	if len(im.entities) != 0 {
		t.Errorf("expected 0 entities after clear, got %d", len(im.entities))
	}
}

func TestIdentityMapSize(t *testing.T) {
	im := newIdentityMap()
	entity := &TestEntity{ID: 1, Name: "Test"}
	im.Put(entity)

	if im.Size() != 1 {
		t.Errorf("expected size 1, got %d", im.Size())
	}
}

func TestQueryBuilderCreation(t *testing.T) {
	qb := QueryBuilder{
		db:        nil,
		tableName: "test_table",
	}
	if qb.tableName != "test_table" {
		t.Errorf("expected table name 'test_table', got %s", qb.tableName)
	}
}

func TestQueryBuilderWhere(t *testing.T) {
	qb := QueryBuilder{}
	qb = qb.Where("name = ?", "test")

	if len(qb.conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(qb.conditions))
	}
	if len(qb.args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(qb.args))
	}
}

func TestQueryBuilderOrderBy(t *testing.T) {
	qb := QueryBuilder{}
	qb = qb.OrderBy("name DESC")

	if len(qb.orderBy) != 1 {
		t.Errorf("expected 1 order by, got %d", len(qb.orderBy))
	}
}

func TestQueryBuilderLimitOffset(t *testing.T) {
	qb := QueryBuilder{}
	qb = qb.Limit(10).Offset(20)

	if qb.limit != 10 {
		t.Errorf("expected limit 10, got %d", qb.limit)
	}
	if qb.offset != 20 {
		t.Errorf("expected offset 20, got %d", qb.offset)
	}
}

func TestQueryBuilderChain(t *testing.T) {
	qb := QueryBuilder{}
	qb = qb.Where("active = ?", true).OrderBy("created_at DESC").Limit(50)

	if len(qb.conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(qb.conditions))
	}
	if len(qb.orderBy) != 1 {
		t.Errorf("expected 1 order by, got %d", len(qb.orderBy))
	}
	if qb.limit != 50 {
		t.Errorf("expected limit 50, got %d", qb.limit)
	}
}
