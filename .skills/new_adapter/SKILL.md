---
name: "new_adapter"
description: "Чеклист добавления нового адаптера: структура, doc.go, Config, интерфейс, тесты"
---
# Adding a New Adapter

## Checklist (execute in order)

1. **Directory structure**: `{adapter_type}/{adapter_name}/`
   - Examples: `queue/nats/`, `db/pg/pgxpool/`, `storage/gcs/`

2. **doc.go**: Create package documentation (see `.skills/doc_go/README.md` for format)

3. **Config struct**: Add `envconfig` tags for all fields
   ```go
   type Config struct {
       Host    string        `envconfig:"MYSERVICE_HOST" default:"localhost"`
       Port    int           `envconfig:"MYSERVICE_PORT" default:"1234"`
       Timeout time.Duration `envconfig:"MYSERVICE_TIMEOUT" default:"10s"`
   }
   ```

4. **Interface**: Implement `Provider` (or `RunableProvider` if runs indefinitely)
   ```go
   type Provider interface {
       Start() error
       io.Closer
   }
   ```

5. **Constructor**: Expose `New(cfg Config)` returning the parent interface
   - If connection required at construction: `NewDefault(cfg Config) (Interface, error)`

6. **OpenTelemetry tracing**: Add spans for all operations
   ```go
   var tracer = otel.Tracer("github.com/pure-golang/adapters/{path}")
   // Span naming: packageName.Operation (e.g., "myadapter.Get")
   ```

7. **Error wrapping**: Wrap all errors with context
   ```go
   return errors.Wrap(err, "failed to connect to MyService")
   ```

8. **README.md**: Usage examples in Russian

9. **Unit tests**: Cover core logic without external services

10. **Integration tests**: Use `testcontainers-go` if external service required
    - See `.skills/integration_testing/README.md` for pattern

11. **Update AGENTS.md**: If new patterns introduced, add references
