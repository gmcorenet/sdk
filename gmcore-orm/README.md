# gmcore-orm

ORM wrapper around GORM with Repository pattern, Unit of Work, and Identity Map.

## Features

- **Repository pattern**: Consistent CRUD operations per entity
- **Unit of Work**: Transactional batch operations
- **Identity Map**: Avoid duplicate entity loading
- **Query Builder**: Chainable query construction
- **YAML configuration**: Configure database from YAML files
- **Multiple drivers**: MySQL, PostgreSQL, SQLite, SQL Server

## Configuration

### YAML Configuration

Create `config/database.yaml` in your app:

```yaml
driver: mysql
dsn: %env(DATABASE_DSN)%

pool:
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 3600
  conn_max_idle_time: 600

auto_migrate: true

logging:
  level: info
  slow_threshold: 1000
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML:

```yaml
dsn: mysql://user:pass@host:3306/db?charset=utf8mb4
```

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-orm"

cfg, err := gmcore_orm.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}

db, err := cfg.Open()
if err != nil {
    log.Fatal(err)
}

orm := gmcore_orm.New(db)
```

## Usage

### Setup

```go
import "github.com/gmcorenet/sdk/gmcore-orm"

// From config
cfg, _ := gmcore_orm.LoadConfig("/opt/gmcore/myapp")
db, _ := cfg.Open()
orm := gmcore_orm.New(db)

// Direct GORM
db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
orm := gmcore_orm.New(db)
```

### Define Entity

```go
type User struct {
    ID    uint
    Name  string
    Email string `gorm:"uniqueIndex"`
}

func (u *User) TableName() string { return "users" }
func (u *User) GetID() interface{} { return u.ID }
```

### Repository Operations

```go
repo := orm.Repository(&User{})

// Create
user := &User{Name: "John", Email: "john@example.com"}
repo.Create(ctx, user)

// Find
user, err := repo.Find(ctx, 1)

// Update
user.Name = "John Doe"
repo.Update(ctx, user)

// Delete
repo.Delete(ctx, user)

// Save (create or update)
repo.Save(ctx, user)
```

### Query Builder

```go
query := orm.Query(&User{})

var users []User
query.Where("name LIKE ?", "%John%").
    OrderBy("created_at DESC").
    Limit(10).
    Find(ctx, &users)

var user User
query.Where("email = ?", "john@example.com").
    First(ctx, &user)
```

### Unit of Work

```go
uow := orm.UnitOfWork()

uow.RegisterNew(&User{Name: "John", Email: "john@example.com"})
uow.RegisterNew(&User{Name: "Jane", Email: "jane@example.com"})

user, _ := repo.Find(ctx, 1)
user.Name = "Modified"
uow.RegisterDirty(user)

if err := uow.Commit(ctx); err != nil {
    // Transaction failed
}
```

### Auto Migration

```go
orm.AutoMigrate(&User{}, &Product{}, &Order{})
```

## Supported Drivers

| Driver   | Import                          |
|----------|--------------------------------|
| MySQL    | `gorm.io/driver/mysql`         |
| PostgreSQL | `gorm.io/driver/postgres`     |
| SQLite   | `gorm.io/driver/sqlite`        |
| SQL Server | `gorm.io/driver/sqlserver`   |

## Configuration Options

| Option              | Type     | Default | Description                |
|--------------------|----------|---------|----------------------------|
| `driver`            | `string` | -       | Database driver            |
| `dsn`               | `string` | -       | Connection string          |
| `pool.max_open_conns` | `int` | `25`     | Max open connections      |
| `pool.max_idle_conns` | `int` | `5`      | Max idle connections      |
| `pool.conn_max_lifetime` | `int` | `3600` | Connection max lifetime (s) |
| `auto_migrate`      | `bool` | `false` | Run auto migration        |
| `logging.level`      | `string` | `info` | Log level                |

## Complete Example

```go
package main

import (
    "context"
    "fmt"

    "github.com/gmcorenet/sdk/gmcore-orm"
)

type User struct {
    ID    uint
    Name  string
    Email string
}

func (u *User) TableName() string { return "users" }
func (u *User) GetID() interface{} { return u.ID }

func main() {
    // Load config and connect
    cfg, err := gmcore_orm.LoadConfig("/opt/gmcore/myapp")
    if err != nil {
        panic(err)
    }

    db, err := cfg.Open()
    if err != nil {
        panic(err)
    }
    defer db.Close()

    orm := gmcore_orm.New(db)

    // Auto migrate
    orm.AutoMigrate(&User{})

    ctx := context.Background()

    // Create
    user := &User{Name: "John", Email: "john@example.com"}
    if err := orm.Repository(&User{}).Create(ctx, user); err != nil {
        panic(err)
    }
    fmt.Printf("Created user with ID: %d\n", user.ID)

    // Find
    found, err := orm.Repository(&User{}).Find(ctx, user.ID)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Found user: %+v\n", found)

    // Query
    var users []User
    orm.Query(&User{}).Where("name LIKE ?", "%John%").Find(ctx, &users)
    fmt.Printf("Found %d users\n", len(users))
}
```
