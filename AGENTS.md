# AGENTS.md — Contexto del proyecto Notifyed

Este archivo le da contexto a cualquier agente de IA (opencode, Claude Code, Cursor, Codex) que trabaje en este repositorio. Leelo siempre antes de tocar código.

## Qué es el proyecto

**Notifyed** es el TP de Programación Web. Un sistema de **avisos** para cursadas universitarias: un profesor crea una cursada, los alumnos se inscriben con su email y reciben notificaciones. Todas las respuestas en este repo están en **español**.

## Stack tecnológico

| Herramienta | Rol |
| --- | --- |
| **Go 1.22** | Backend. Solo stdlib: `net/http` (sin frameworks). |
| **PostgreSQL 18** | Base de datos, corre en Docker (`postgres:18-alpine`). |
| **sqlc v2** | Genera el código Go tipo-safe desde queries SQL. **No escribir código SQL a mano en Go**: se escribe en `.sql` y se regenera. |
| **lib/pq** | Driver PostgreSQL para Go. |
| **google/uuid** | IDs UUID para claves primarias y tests. |
| **Docker / docker-compose** | Levanta la base + el web server. |
| **Makefile** | Orquesta todo el flujo (build, up, test, clean). |

## Estructura del repo

```
db/
├── schema/schema.sql       # DDL de la base. Tablas: courses, semesters, users, inscriptions, notices.
├── queries/*.sql           # Queries SQL fuente para sqlc (users, courses, semesters, inscriptions, notices).
└── sqlc/                   # CÓDIGO GENERADO por sqlc. NO editar a mano: se regenera con `make generate`.
    ├── models.go
    ├── *.sql.go            # Funciones tipo-safe generadas.
    └── *_test.go           # Tests de las queries (los únicos tests que hay hoy).
handlersTP/                 # Handlers HTTP (package handlersTP).
static/                     # Frontend estático (index.html y assets).
main.go                     # Punto de entrada: registra las rutas y levanta :8080.
```

## Tablas y relaciones (clave para trabajar con queries)

- **users**: `id_user` (uuid), `fullname`, `email` (UNIQUE), `password_hash`.
- **courses**: `id_course` (uuid), `name`. Las materias.
- **semesters**: una cursada. `id_semester`, `id_course` (FK→courses), `professor_id` (FK→users), `year`, `sub_token` (UNIQUE, lo que resuelve el QR), `end_date`. **Una cursada está cerrada si `end_date < now()`** (se calcula en la query, no hay columna `is_open`).
- **inscriptions**: `id_inscription`, `id_semester` (FK→semesters), `email`, `active` (por defecto true), `created_at`. UNIQUE(email, id_semester).
- **notices**: `id_notice`, `id_semester`, `id_sender` (FK→users), `title`, `message`, `created_at`.

Regla de oro de las queries: las cursadas "vigentes" se filtran con `end_date >= CURRENT_DATE` o `> now()`; las "cerradas/vencidas" con lo opuesto. El campo calculado `is_open` y la cuenta `active_subscribers` son **alias SQL** (ver `semesters_queries.sql`), no columnas.

## Comandos del Makefile (flujo principal)

```
make            # all: build + up + test + clean de corrido (el flujo que usa el profe)
make generate   # regenera db/sqlc/ con sqlc (¡siempre que cambies db/queries/*.sql!)
make up         # docker compose up la base y espera a que esté saludable
make down       # baja los contenedores y borra los volúmenes (¡borra datos!)
make test       # go test -v ./db/sqlc (requiere la base arriba y el .env)
make clean      # borra el binario temporal
```

### Cómo correr los tests (nuevo en este repo)

1. `cp .env.example .env` (si no existe).
2. `make up` para levantar PostgreSQL.
3. `make test` para correr los tests de queries.
4. (Opcional) `make down` para bajar todo.

**Importante**: los tests usan `TEST_DB_URL` que arma el Makefile desde el `.env`. Si inventás un DSN a mano, que coincida. El esquema se aplica solo la primera vez que el contenedor se levanta (montado en `/docker-entrypoint-initdb.d`); `make down` borra el volumen, así que si cambiás `schema.sql`, bajá y levantá de nuevo.

## Convenciones del código

- **sqlc**: cada query en `db/queries/<tabla>_queries.sql` con el comentario magico `-- name: NombreMetodo :one|:many|:execrows`. Regenerar con `make generate` después de tocar esos archivos. Los `:one` devuelven `sql.ErrNoRows` si no matchean nada; `:execrows` devuelve filas afectadas.
- **Tests de queries** (`db/sqlc/*_test.go`):
  - Corren contra una Postgres real levantada por Docker (no mocks, no sqlite).
  - Cada test usa `withTx(t)` para abrir una transacción que se revierte al final (los tests no se pisan entre sí).
  - Helpers de setup en `main_test.go`: `crearProfesor`, `crearMateria`, `crearCursada`, `crearCursadaVigente`. Usarlos en vez de insertar a mano.
  - `emailUnico("prefijo")` evita chocar con la UNIQUE de usuarios.
  - Estilo: tabla de casos con campo `wantErr`, subtests con `t.Run`, comentarios en español, seccionadores `// ----` para separar cada query.
  - La fecha se castea con `Format("2006-01-02")` (layout de Go para YYYY-MM-DD).
- **Handlers** (`handlersTP/`): funciones `func Nombre(w http.ResponseWriter, r *http.Request)` registradas en `main.go` con `http.HandleFunc`. Sin frameworks.
- **Errores HTTP**: mensajes en español.
- **Nombres**: identificadores en inglés (el código), comentarios y mensajes en español.

## Ramas y flujo de trabajo

- `main`: rama principal remota.
- `tp1` / `tp2`: entregas anteriores. La README indica que el flujo del TP vive en `tp2`.
- La rama **actual de desarrollo** (donde se está aprendiendo) es `rochatomas-ing`.

Regla: no pushear a `main` directo. Los cambios sobre el TP van en la rama de trabajo correspondiente.

## Secretos y seguridad

- `.env` tiene las credenciales y está en `.gitignore`. Nunca committear `.env` ni credenciales reales. Usar `.env.example` como plantilla.
- `password_hash` se guarda hasheado (bcrypt, prefijo `$2a$`); nunca loguear ni devolver hashes en respuestas.

## Checklist antes de dar por cerrada una tarea

1. ¿Cambiaste `db/queries/`? Correr `make generate` y commitear también `db/sqlc/`.
2. ¿Cambiaste `schema.sql`? Probar con `make down && make up` (el esquema solo se aplica al crear el volumen).
3. ¿Escribiste código nuevo? Correr `make test` (y `go vet .`/`go build` si tocaste Go fuera de db/sqlc).
4. ¿Los tests pasan? Confirmado antes de declarar la tarea terminada.