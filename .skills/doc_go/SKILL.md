---
name: "doc_go"
description: "Требования к doc.go: формат, обязательные секции, синхронизация с кодом"
---
# doc.go Requirements

Every package and sub-package **must** have a `doc.go` with a package-level comment containing:

1. **What interface it implements** and from which parent package
2. **Entry point** — which constructor to use
3. **Env variables** — list with defaults
4. **Constraints** — thread-safety, required `Close()`, ordering requirements, etc.

## Template
```go
// Package {name} implements the {parent}.{Interface} interface for {Service}.
//
// Usage:
//
//	instance := {name}.New(cfg)
//	defer instance.Close()
//
// Configuration:
//
//	{ENV_VAR_1}  — description (default: value)
//	{ENV_VAR_2}  — description (default: value)
//
// Constraints:
//
//	- Thread-safe: yes/no
//	- Requires Close() to release resources
//	- {other constraints}
package {name}
```

## Real Example
```go
// Package redis implements the kv.Store interface for Redis.
//
// Usage:
//
//	store := redis.New(cfg)
//
// Configuration:
//
//	REDIS_ADDR     — server address (default: localhost:6379)
//	REDIS_PASSWORD — password
//	REDIS_DB       — database number (default: 0)
package redis
```

## Rules
- `doc.go` is the **package contract** — must be kept in sync with code
- Reviewed on every PR
- **When working in a package directory, read its `doc.go` first** to optimize inference context
- Comments in Russian (package-level docs in Russian per project convention)
