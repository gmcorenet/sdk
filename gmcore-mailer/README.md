# gmcore-mailer

Email sending library for gmcore applications with SMTP support.

## Features

- **SMTP mailer**: Send emails via SMTP server
- **Memory mailer**: Store emails in memory (for testing)
- **Attachments**: Support for file attachments
- **HTML emails**: HTML and plain text email support
- **YAML configuration**: Configure mailer from YAML files

## Configuration

### YAML Configuration

Create `config/mailer.yaml` in your app:

```yaml
host: %env(SMTP_HOST)%
port: 587
username: %env(SMTP_USER)%
password: %env(SMTP_PASS)%
from: %env(MAILER_FROM)%
from_name: My App
reply_to: noreply@example.com
encryption: tls
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML:

```yaml
host: %env(SMTP_HOST)%
username: %env(SMTP_USER)%
password: %env(SMTP_PASS)%
```

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-mailer"

cfg, err := gmcore_mailer.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}

mailer := gmcore_mailer.NewSMTPMailer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
```

## Usage

### Basic Email

```go
import "github.com/gmcorenet/sdk/gmcore-mailer"

mailer := gmcore_mailer.NewSMTPMailer("smtp.example.com", 587, "user", "pass")

email := gmcore_mailer.NewEmail(
    "sender@example.com",
    "recipient@example.com",
    "Subject",
    "Hello, World!",
)

err := mailer.Send(email)
if err != nil {
    log.Fatal(err)
}
```

### Email Builder

```go
email := gmcore_mailer.NewEmail("", "", "", "")

email.From = "sender@example.com"
email.AddTo("recipient1@example.com")
email.AddTo("recipient2@example.com")
email.AddCc("cc@example.com")
email.AddBcc("bcc@example.com")
email.Subject = "Subject"
email.Body = "Message body"
email.SetHtml(true)

// Add attachment
email.AddAttachment("file.pdf", fileContent)

err := mailer.Send(email)
```

### Memory Mailer (Testing)

```go
mailer := gmcore_mailer.NewMemoryMailer()

// Send emails
mailer.Send(email1)
mailer.Send(email2)

// Get all sent emails
emails := mailer.GetEmails()
for _, e := range emails {
    fmt.Printf("To: %s, Subject: %s\n", e.To, e.Subject)
}

// Clear mailbox
mailer.Clear()
```

## Configuration Options

| Option      | Type     | Default | Description                    |
|-------------|----------|---------|--------------------------------|
| `host`       | `string` | -       | SMTP server hostname           |
| `port`       | `int`    | `587`   | SMTP port                      |
| `username`   | `string` | -       | SMTP username                  |
| `password`   | `string` | -       | SMTP password                  |
| `from`       | `string` | -       | Default from address           |
| `from_name`  | `string` | -       | Default from name              |
| `reply_to`   | `string` | -       | Reply-To address              |
| `encryption` | `string` | `tls`   | Encryption (tls/ssl/none)     |

## Email Structure

```go
type Email struct {
    From        string
    To          []string
    Cc          []string
    Bcc         []string
    Subject     string
    Body        string
    Html        bool
    Attachments []Attachment
}

type Attachment struct {
    Name    string
    Content []byte
}
```

## Complete Example

```go
package main

import (
    "log"

    "github.com/gmcorenet/sdk/gmcore-mailer"
)

func main() {
    // Load config
    cfg, err := gmcore_mailer.LoadConfig("/opt/gmcore/myapp")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Create mailer
    mailer := gmcore_mailer.NewSMTPMailer(
        cfg.Host,
        cfg.Port,
        cfg.Username,
        cfg.Password,
    )

    // Build email
    email := gmcore_mailer.NewEmail(
        cfg.From,
        "recipient@example.com",
        "Welcome to My App",
        "Hello!\n\nWelcome to our application.",
    )
    email.SetHtml(false)

    // Send
    if err := mailer.Send(email); err != nil {
        log.Fatalf("Failed to send email: %v", err)
    }

    log.Println("Email sent successfully!")
}
```
