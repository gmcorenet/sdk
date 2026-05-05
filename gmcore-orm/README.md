# gmcore-orm

Wrapper ORM around GORM with Repository pattern, Unit of Work, and Identity Map.

## Installation

```bash
go get github.com/gmcorenet/sdk/gmcore-orm
```

## Usage

```go
import "github.com/gmcorenet/sdk/gmcore-orm"

// Define an entity
type User struct {
    ID   uint
    Name string
}

func (u *User) TableName() string { return "users" }
func (u *User) GetID() interface{} { return u.ID }

// Setup ORM
orm := gmcore_orm.New(db)
orm.AutoMigrate(&User{})

// Using Repository
repo := orm.Repository(&User{})
repo.Create(ctx, &User{Name: "John"})
user, _ := repo.Find(ctx, 1)

// Using Unit of Work
uow := orm.UnitOfWork()
uow.RegisterNew(&User{Name: "Jane"})
uow.Commit(ctx)

// Using Query Builder
orm.Query(&User{}).Where("name LIKE ?", "%John%").Find(ctx, &users)

// Using Identity Map
im := orm.IdentityMap()
im.Put(user)
retrieved, ok := im.Get("users", 1)
```

## Features

- **Repository Pattern**: Consistent CRUD operations per entity
- **Unit of Work**: Transactional batch operations
- **Identity Map**: In-memory entity caching
- **Query Builder**: Chainable query construction
- **Registry**: Centralized entity registration

## License

MIT
