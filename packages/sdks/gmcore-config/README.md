# Config SDK

Sistema de configuración YAML con soporte para variables de entorno y parámetros.

## Funciones Principales

### `LoadYAML(path string, out interface{}, options Options) error`
Carga y parsea un archivo YAML, resolviendo placeholders de entorno y parámetros.

**Options:**
```go
type Options struct {
    Env        map[string]string  // Variables de entorno
    Parameters map[string]string  // Parámetros de configuración
    Strict     bool               // Error si falta placeholder
}
```

### `ResolveString(content string, options Options) (string, error)`
Resuelve placeholders en strings:
- `%env(NAME)%` - Variables de entorno
- `%parameter.KEY%` - Parámetros

### `ParseEnvFile(path string) map[string]string`
Parsea un archivo `.env` retornando un mapa de clave=valor.

### `LoadEnvFiles(paths ...string) map[string]string`
Carga y mergea múltiples archivos `.env`.

### `LoadAppEnv(appPath string) map[string]string`
Carga configuración de entorno para una aplicación buscando:
- `.env`
- `.env.local`
- `config/{appname}.env`

Aplica prefijos `APP_` a variables específicas.

### `EnvList(values map[string]string) []string`
Convierte mapa de entorno a lista de strings "KEY=value" ordenados.

## Uso

```go
env := gmcoreconfig.LoadAppEnv("/path/to/app")
config := &MyConfig{}
err := gmcoreconfig.LoadYAML("config.yaml", config, gmcoreconfig.Options{
    Env:    env,
    Strict: true,
})
```
