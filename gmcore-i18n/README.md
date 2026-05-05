# gmcore-i18n

Internationalization (i18n) library for gmcore applications with translation management.

## Features

- **Translation catalogs**: YAML-based translation files
- **Pluralization**: ICU plural rules support
- **Parameter interpolation**: Dynamic message content
- **Domain support**: Organize translations by domain
- **Locale resolution**: Automatic fallback handling
- **Frontend integration**: JSON payloads for JavaScript

## Configuration

### YAML Configuration

Create `config/i18n.yaml` in your app:

```yaml
default_locale: en
fallback_locale: en

directories:
  - translations/en
  - translations/es
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML:

```yaml
default_locale: %env(DEFAULT_LOCALE)%
```

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-i18n"

cfg, err := gmcore_i18n.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}

translator, err := cfg.Build()
if err != nil {
    log.Fatal(err)
}
```

## Translation Files

Translation files are YAML with dot-notation keys:

```yaml
# translations/en/messages.yaml
greeting: "Hello, World!"
welcome: "Welcome, {name}!"
items_count: "{count, plural, =0 {No items} =1 {One item} other {# items}}"
```

## Usage

### Basic Translation

```go
translator := gmcore_i18n.NewTranslator()

// Simple translation
msg := translator.T("en", "hello")
// Returns: "Hello"

msg = translator.T("en", "greeting", gmcore_i18n.Params{"name": "John"})
// Returns: "Hello, John!"
```

### Domain-Scoped Translation

```go
// domain:key format
msg := translator.T("en", "emails:welcome")
msg := translator.T("en", "errors:not_found")
```

### Pluralization

```go
// ICUVariant format
msg := translator.TC("en", "items_count", 0)
// Returns: "No items"

msg := translator.TC("en", "items_count", 1)
// Returns: "One item"

msg := translator.TC("en", "items_count", 5)
// Returns: "5 items"
```

### Loading Translations

```go
// Load from directory
translator, err := gmcore_i18n.LoadDir("translations", "en")

// Load from multiple directories
translator, err := gmcore_i18n.LoadDirs([]string{
    "translations/en",
    "translations/es",
}, "en")
```

## Parameters

Interpolation supports multiple formats:

```yaml
# {{variable}}
# {variable}
# %variable%

message: "Hello, {name}!"
```

## Locale Resolution

```go
// Automatic fallback
translator.ResolveLocale("en-US")  // Returns "en" if "en-US" not found
translator.ResolveLocale("es-MX")  // Returns "es" if "es-MX" not found
translator.ResolveLocale("fr")      // Returns fallback if "fr" not found
```

## Frontend Integration

```go
// Generate JSON for frontend
payload := translator.FrontendPayload("en", "domain=messages")

// Returns FrontendPayload struct with:
// - Locale
// - Messages map
// - Hash for cache busting
```

## Configuration Options

| Option           | Type     | Default | Description                |
|------------------|----------|---------|----------------------------|
| `default_locale`  | `string` | `en`    | Default locale              |
| `fallback_locale` | `string` | `en`    | Fallback locale             |
| `directories`     | `[]string` | -      | Translation directories     |

## Directory Structure

```
translations/
├── en/
│   └── messages.yaml
├── es/
│   └── messages.yaml
└── fr/
    └── messages.yaml
```

## Complete Example

```go
package main

import (
    "fmt"

    "github.com/gmcorenet/sdk/gmcore-i18n"
)

func main() {
    // Load translations
    translator, err := gmcore_i18n.LoadDir("translations", "en")
    if err != nil {
        panic(err)
    }

    // Translate
    fmt.Println(translator.T("en", "hello"))
    fmt.Println(translator.T("en", "greeting", gmcore_i18n.Params{"name": "John"}))

    // Plural
    fmt.Println(translator.TC("en", "items", 0))
    fmt.Println(translator.TC("en", "items", 1))
    fmt.Println(translator.TC("en", "items", 5))

    // Check support
    fmt.Println("Supported:", translator.SupportedLocales())
}
```
