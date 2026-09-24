---
name: notifyed-tooling
description: Recetas para buildear, testear y debuggear este repo (Go, sqlc, Docker, Makefile). Usar cuando el usuario hable de "make", "sqlc", "tests", "docker", "build", "regenerar", "bajar/levantar la base" o necesite arreglar el flujo de desarrollo de Notifyed.
---

# Notifyed Tooling

Recetas operativas para trabajar con este stack. Combinar siempre con `AGENTS.md` (el contexto del repo).

## Flujo completo del profe

```bash
make       # build + up + test + clean de corrido
```

Equivale a: `make build` → `make down` → `make up` → `make test` → `make clean`.

## Levantar y testear (por pasos)

```bash
cp .env.example .env          # solo la primera vez
make up                       # levanta PostgreSQL en Docker y espera a que esté saludable
make test                     # go test -v ./db/sqlc (requiere la base arriba)
make down                     # para y BORRA los volúmenes (¡perdés datos!)
```

Requiere `TEST_DB_URL`, que el Makefile arma solo desde el `.env`. No inventar DSNs.

## Ciclo de sqlc (cuando cambiás una query)

```bash
make generate                 # regenera todo db/sqlc/ desde db/queries/*.sql
make test                     # confirma que lo generado compila y los tests pasan
```

Regla: **`db/sqlc/` nunca se edita a mano**. Si una query cambia, se escribe en `db/queries/` y se regenera.

## Si cambiás el esquema (schema.sql)

El DDL solo se aplica la primera vez que se crea el volumen. Después de tocar `schema.sql`:

```bash
make down && make up          # volumen nuevo ⇒ el esquema se re-aplica desde cero
make test
```

## Debug común

| Síntoma | Causa probable | Fix |
| --- | --- | --- |
| `make test` dice "la base no responde" | Contenedor apagado o `.env` ausente | `make up`; `cp .env.example .env` |
| Tests fallan con constraints UNIQUE | Datos de sesiones previas | `make down && make up` para resetear |
| `make test` compila pero "not implemented" | Query con `:one` esperando fila y no hay | Fijarse en `sql.ErrNoRows` |
| `sqlc generate` tira error | Query mal escrita o falta el comentario `-- name:` | Leer `db/queries/` y corregir la sintaxis |

## Orden sugerido si algo se rompe

1. `make up` (¿la base está arriba?)
2. `make generate` (¿está sincronizado `db/sqlc/` con `db/queries/`?)
3. `make test` (¿compila y los tests pasan?)
4. Si tocaste Go fuera de `db/sqlc`: `go vet .` y `go build .`