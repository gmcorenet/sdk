# gmcore-transport

Unified transport layer for gmcore apps supporting both TCP and Unix Domain Sockets.

## Features

- **Multiple transport modes**: UDS, TCP, or Both
- **Built-in security**: HMAC signature and mutual authentication support
- **Automatic pairing**: App-to-gateway secure pairing
- **Lifecycle commands**: Start, stop, restart, reload, status via UDS
- **Pluggable security**: Use HMAC, certificates, or custom security providers

## Quick Start

### Server (App)

```go
import "github.com/gmcorenet/sdk/gmcore-transport"

// Create transport
t := gmcore_transport.New(gmcore_transport.Config{
    Mode: gmcore_transport.ModeUDS,
    Path: "var/socket/myapp.sock",
})

// Add lifecycle handlers
t.Lifecycle().OnStart(func() error { /* start app */ return nil })
t.Lifecycle().OnStop(func() error { /* stop app */ return nil })
t.Lifecycle().OnRestart(func() error { /* restart app */ return nil })
t.Lifecycle().OnStatus(func() (map[string]any, error) {
    return map[string]any{"status": "running"}, nil
})

// Listen
ctx := context.Background()
t.Listen(ctx)
```

### Client (Gateway)

```go
// Connect to app UDS
client := gmcore_transport.NewClient("unix", "/opt/gmcore/myapp/var/socket/myapp.sock")
client.UseSecurity(gmcore_transport.NewHMACSecurity(secret))

if err := client.Connect(); err != nil {
    log.Fatal(err)
}

// Send lifecycle command
resp, err := client.Command("restart", nil)
```

## Configuration

### Mode Options

- `ModeUDS` - Unix Domain Socket only
- `ModeTCP` - TCP/IP only
- `ModeBoth` - Both UDS and TCP

### Security Providers

```go
// No security (development)
t.UseSecurity(&gmcore_transport.NoOpSecurity{})

// HMAC signature
t.UseSecurity(gmcore_transport.NewHMACSecurity([]byte("shared-secret")))

// Mutual authentication with certificates
sec, err := gmcore_transport.NewMutualSecurity("var/keys")
t.UseSecurity(sec)
```

## Pairing

Apps automatically pair with gateway on first connection:

```go
pm := gmcore_transport.NewPairingManager(appID, appName, "var/keys")
if err := pm.RequestPairing("gateway-host"); err != nil {
    log.Fatal(err)
}
```

## Lifecycle Commands

| Command   | Description                    |
|-----------|--------------------------------|
| `start`   | Start the application          |
| `stop`    | Stop the application           |
| `restart` | Restart the application        |
| `reload`  | Reload configuration           |
| `status`  | Get application status        |

## Socket Permissions

Default permissions: `0660` (owner and group read/write)

The socket is created in `var/socket/` directory. Gateway must be in the same group as apps to access sockets.

## Directory Structure

```
var/
├── socket/
│   └── app.sock      # UDS socket
└── keys/
    ├── cert.pem      # Certificate
    ├── key.pem       # Private key
    ├── pairing.json  # Gateway pairing info
    └── peers/        # Peer certificates
```
