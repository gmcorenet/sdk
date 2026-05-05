# gmcore-security

Security components for gmcore applications including authentication, authorization, and password hashing.

## Features

- **Password hashing**: BCrypt hasher with configurable cost
- **Authentication**: Basic Auth authenticator
- **Authorization**: Role-based access control with voters
- **YAML configuration**: Configure security from YAML files
- **Security checker**: Centralized access control

## Configuration

### YAML Configuration

Create `config/security.yaml` in your app:

```yaml
role_prefix: "ROLE_"
default_role: "ROLE_USER"
password_cost: 10

firewall:
  enabled: true
  patterns:
    - ^/admin
  excludes:
    - ^/health
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML:

```yaml
password_cost: %env(PASSWORD_COST)%
```

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-security"

cfg, err := gmcore_security.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}

// Use config
hasher := gmcore_security.NewBCryptHasher(cfg.PasswordCost)
```

## Usage

### Password Hashing

```go
import "github.com/gmcorenet/sdk/gmcore-security"

hasher := gmcore_security.NewBCryptHasher(10)

// Hash password
hash, err := hasher.Hash("mysecretpassword")
if err != nil {
    log.Fatal(err)
}

// Verify password
if hasher.Verify(hash, "mysecretpassword") {
    // Password correct
}

// Check if needs rehash
if hasher.NeedsRehash(hash) {
    // Rehash needed
}
```

### Basic Authentication

```go
hasher := gmcore_security.NewBCryptHasher(10)
auth := gmcore_security.NewBasicAuthenticator("My App", hasher)

// Add users (password should be pre-hashed)
auth.AddUser("admin", "$2a$10$...")

// Authenticate request
user, err := auth.Authenticate(r)
if err != nil {
    auth.OnAuthFailure(w, r, err)
    return
}

auth.OnAuthSuccess(w, r, user)
```

### Authorization

```go
checker := gmcore_security.NewSecurityChecker()

// Add role voter
roleVoter := gmcore_security.NewRoleVoter("ROLE_")
checker.AddVoter(roleVoter)

// Check access
user := gmcore_security.NewSimpleUser("john", hash, []string{"ROLE_ADMIN", "ROLE_USER"})

if checker.IsGranted(user, "EDIT", nil) {
    // User can edit
}
```

### Role Voter

The role voter checks if user has the required role:

```go
voter := gmcore_security.NewRoleVoter("ROLE_")

user := NewSimpleUser("john", hash, []string{"ROLE_ADMIN"})
result := voter.Vote(user, "ADMIN", nil)
// Returns ACCESS_GRANTED

result = voter.Vote(user, "SUPERADMIN", nil)
// Returns ACCESS_DENIED
```

### User Interface

```go
type User interface {
    GetIdentifier() interface{}
    GetRoles() []string
    GetPasswordHash() string
    EraseCredentials()
}
```

### Simple User

```go
user := gmcore_security.NewSimpleUser(
    "john@example.com",
    "$2a$10$...",
    []string{"ROLE_USER", "ROLE_ADMIN"},
)

fmt.Println(user.GetIdentifier()) // john@example.com
fmt.Println(user.GetRoles())      // [ROLE_USER ROLE_ADMIN]
```

## Security Checker

The security checker collects multiple voters and makes authorization decisions:

```go
checker := gmcore_security.NewSecurityChecker()

checker.AddVoter(roleVoter1)
checker.AddVoter(roleVoter2)

if checker.IsGranted(user, "EDIT", someObject) {
    // Access granted
} else {
    // Access denied
}
```

## Access Constants

```go
const (
    ACCESS_GRANTED = 1  // Voter grants access
    ACCESS_ABSTAIN = 0  // Voter abstains from decision
    ACCESS_DENIED  = -1 // Voter denies access
)
```

## Context Helpers

Store and retrieve user from context:

```go
// Save user to context
ctx := gmcore_security.SaveUserToContext(request.Context(), user)

// Retrieve user from context
user := gmcore_security.UserFromContext(ctx)
if user != nil {
    // User is authenticated
}
```

## Configuration Options

| Option          | Type     | Default   | Description                |
|-----------------|----------|-----------|----------------------------|
| `role_prefix`   | `string` | `ROLE_`   | Prefix for role checking   |
| `default_role`   | `string` | `ROLE_USER` | Default role for users     |
| `password_cost`  | `int`    | `10`      | BCrypt cost factor         |
| `firewall.enabled` | `bool` | `true`    | Enable firewall            |

## Complete Example

```go
package main

import (
    "net/http"

    "github.com/gmcorenet/sdk/gmcore-security"
)

func main() {
    // Setup security
    hasher := gmcore_security.NewBCryptHasher(10)
    auth := gmcore_security.NewBasicAuthenticator("My App", hasher)

    // Create hashed password (do this once, store hash)
    hash, _ := hasher.Hash("admin123")
    auth.AddUser("admin", hash)

    checker := gmcore_security.NewSecurityChecker()
    checker.AddVoter(gmcore_security.NewRoleVoter("ROLE_"))

    // Use in HTTP handler
    http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
        user, err := auth.Authenticate(r)
        if err != nil {
            auth.OnAuthFailure(w, r, err)
            return
        }

        if !checker.IsGranted(user, "ADMIN", nil) {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }

        w.Write([]byte("Welcome, admin!"))
    })
}
```
