# AGENTS.md — Contexto del proyecto Notifyed

Este archivo le da contexto a cualquier agente de IA (opencode, Claude Code, Cursor, Codex) que trabaje en este repositorio. Leelo siempre antes de tocar código.

## Qué es el proyecto

**Notifyed** es el TP de Programación Web. Un sistema de **avisos** para cursadas universitarias. Todas las respuestas en este repo están en **español**.

### Los dos actores (importante para no inventar funcionalidad)

| | **Alumno** | **Profesor** |
| --- | --- | --- |
| ¿Tiene cuenta? | **No.** Sólo su email en `inscriptions` | **Sí.** Es una fila de `users` |
| ¿Qué hace? | **Sólo suscribirse o darse de baja** de una cursada | **Usa todo el sistema** |
| ¿Cómo entra? | Escanea el QR / `sub_token` | Email y password |
| ¿Qué ve? | Nada: **no hay panel, ni perfil, ni lista de avisos** | Materias, cursadas, avisos, inscriptos |
| ¿Cómo le llegan los avisos? | Por **email** (el envío real es un TP posterior, no existe todavía) | Los publica él desde el sistema |

**Consecuencias prácticas:**

- La tabla **`users` es exclusiva del profesor**. El alumno nunca tiene fila ahí.
- No hay endpoints para "mis avisos" del alumno, ni panel de perfil: **no los implementes aunque parezcan obvios**.
- La parte del alumno es mínima: alta y baja de inscripción.
- `notices.id_sender` siempre apunta a un `users` (un profesor), nunca a un alumno.

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

### Lo que existe hoy

```
db/
├── schema/schema.sql       # DDL de la base. Tablas: courses, semesters, users, inscriptions, notices.
├── queries/*.sql           # Queries SQL fuente para sqlc (users, courses, semesters, inscriptions, notices).
└── sqlc/                   # CÓDIGO GENERADO por sqlc. NO editar a mano: se regenera con `make generate`.
    ├── models.go
    ├── *.sql.go            # Funciones tipo-safe generadas.
    └── *_test.go           # Tests de las queries (los únicos tests que hay hoy).
handlersTP/                 # Handlers HTTP (package handlersTP). Hoy sólo indexHandler.go.
static/                     # Frontend estático (index.html y assets).
main.go                     # Punto de entrada: registra las rutas y levanta :8080.
Makefile                    # Orquestación: build, up, down, test, generate.
docker-compose.yml          # Servicios `web` y `database`.
```

### Lo que se agrega en el TP3 (capa HTTP)

Estos directorios **aún no existen**: se crean durante el TP3. Respetá la separación cuando los agregues.

```
db_conn/                    # Conexión a la BD y armado del DSN desde variables de entorno.
handlersTP/                 # Capa HTTP. Lo único que toca ResponseWriter y Request.
  httputil.go               # Helpers compartidos: WriteJSON, HandleError, ParseUUID.
middleware/                 # Transversal a todas las rutas (logging).
service/                    # Reglas de negocio. NO sabe que existe HTTP.
dto/                        # Los JSON que entran y salen. Ocultan password_hash.
requests.sh                 # Evidencia de pruebas de la API con curl.
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

## TP3: la API REST

Consigna: `docs/tp3.pdf`, sección *"Trabajo de Cursada: Exponiendo la Lógica como API"*. Pide (1) conectar la BD e instanciar el repositorio de sqlc, (2) handlers CRUD usando los métodos `Create…/List…/Get…/Update…/Delete…`, (3) probar con curl o HURL en un `requests.sh`, (4) instrucciones de ejecución y Docker.

### Las tres capas (diapositiva 11)

Cada capa habla **sólo con la de abajo**.

| Capa | Responsabilidad | **Nunca** debe contener |
| --- | --- | --- |
| `handlersTP/` | Interpretar HTTP: método, ID de la URL, decodificar body, elegir status, escribir la respuesta | Reglas de negocio, SQL, acceso a la BD |
| `service/` | Validación y reglas del dominio. Orquestar operaciones | `http.ResponseWriter`, `*sql.DB`, SQL |
| `db/sqlc/` (generado) | Persistencia. Ya está hecho y testeado | — |

El service se puede testear **sin levantar HTTP ni base de datos** (diapositiva 8). Esa es la ventaja de la separación.

### El patrón de ruteo (diapositivas 26-27)

Un handler por ruta base, que rutea internamente por método HTTP:

```go
// /notices  ->  GET (listar) | POST (crear)
func noticesHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        listNotices(w, r)
    case http.MethodPost:
        createNotice(w, r)
    default:
        httputil.HandleError(w, http.StatusMethodNotAllowed, "metodo no permitido")
    }
}

// /notices/{id}  ->  GET | PUT | DELETE
func noticeHandler(w http.ResponseWriter, r *http.Request) { /* switch igual */ }
```

Nombres de rutas en **inglés** (`/notices`), **mensajes de error en español** (convención del repo).

### Traducción de errores (diapositiva 13)

*"El handler decide si un error se responde como 400, 404, 409 o 500."*

| Situación | Status |
| --- | --- |
| Falta un campo obligatorio; `title` o `message` vacío | `400` |
| El UUID de la URL no se pudo parsear | `400` |
| El body no es JSON válido | `400` |
| El recurso no existe (`sql.ErrNoRows`) | `404` |
| Método HTTP no soportado en esa ruta | `405` |
| Violación de UNIQUE en `users` o `courses` | `409` |
| Cualquier otro error (base, red) | `500` + log |

> **Ojo:** en `inscriptions` el `409` **no aplica**. `CreateInscription` ya trae `ON CONFLICT DO UPDATE SET active = true`, así que re-inscribirse **reactiva** y devuelve `201`.

### Reglas de la API

- **Siempre JSON**, nunca texto plano. `Content-Type: application/json`.
- Los errores también en JSON: `{"error": "mensaje en español"}`.
- **`password_hash` nunca sale.** El model `User` de sqlc tiene `json:"password_hash"`: usá un DTO propio (`dto/`) sin ese campo. Es una garantía, no una precaución.
- Las decisiones entre endpoints van en `docs/api-contrato.md`, **no** en la cabeza de cada uno.
- Sin autenticación en este TP: la consigna no la pide. No agregues login ni sesiones.

### Endpoints del MVP

**Del alumno** (mínimo, sólo esto):

```
POST   /semesters/{sub_token}/inscriptions    body {"email": "..."}   -> 201
DELETE /semesters/{sub_token}/inscriptions    body {"email": "..."}   -> 204 | 404
GET    /semesters/{sub_token}/inscriptions?email=...                 -> estado de la suscripción
```

**Del profesor**:

```
POST   /users | /courses | /semesters | /notices                     -> 201
GET    /notices | /notices/{id} | /semesters/{id}/inscriptions       -> 200
GET    /semesters/{sub_token}                                       -> 200 | 404
GET    /semesters/{id}/notices                                      -> historial publicado (ListNoticesWithSender)
GET    /profesores/{id}/semesters                                   -> ListSemestersByProfessor
GET    /semesters/{id}/suscriptos                                   -> CountActiveBySemester
POST   /inscriptions/{id}/toggle                                    -> SetInscriptionActive
PUT    /semesters/{id}                                              -> UpdateSemesterEndDate (cerrar cursada)
PUT    /notices/{id} | DELETE /notices/{id} | DELETE /courses/{id}   -> 200 | 204
```

### Queries que ya resuelven el modelo del alumno

No hace falta tocar `db/queries/` ni el esquema para la parte del alumno:

| Query | Para qué |
| --- | --- |
| `GetSemesterByToken` | Resolver `{sub_token}` → `id_semester` |
| `GetInscriptionByEmailAndSemester` | Buscar por email. `ErrNoRows` → **404** si nunca se inscripto |
| `SetInscriptionActive(id, email, false)` | **La baja.** Un solo update sirve para baja y reactivación |
| `CreateInscription` | El alta. Reactiva en vez de fallar si ya estaba |
| `ListActiveEmailsBySemester` | Destinatarios de un aviso. **Ya está lista para el envío de emails del TP futuro** |

### Los `Update…` de sqlc son angostos

**No existe un "update del objeto completo".** Cada query actualiza columnas puntuales:

| Query | Actualiza |
| --- | --- |
| `UpdateCourseName` | sólo el nombre |
| `UpdateNoticeMessage` | **sólo el mensaje, no el título** |
| `UpdateUserFullname` | sólo el nombre |
| `UpdateUserPassword` | sólo el password |
| `UpdateSemesterEndDate` | sólo `end_date` |
| `SetInscriptionActive` | sólo `active` |

Si un endpoint necesita cambiar más campos, hace falta una query nueva: **se pide, no se escribe** (ver el reparto más abajo).

## Reparto de archivos entre los dos

**Regla de oro: cada archivo tiene un dueño. Si no es tuyo, no lo tocás; si lo necesitás, lo pedís.**

| Archivo | Dueño |
| --- | --- |
| `main.go` | **A** |
| `Makefile`, `docker-compose.yml`, `.env.example` | **A** |
| `db/queries/**`, `db/sqlc/**` | **A** |
| `AGENTS.md` | **A** |
| `README.md` | **B** |
| `requests.sh` | **B** lo crea; cada uno agrega su sección |
| `db_conn/conn.go` | **A** |
| `handlersTP/httputil.go`, `middleware/`, `dto/` | **B** |
| `handlersTP/noticesHandler.go`, `inscriptionsHandler.go` (+ sus `service/`) | **B** |
| `handlersTP/usersHandler.go`, `coursesHandler.go`, `semestersHandler.go` (+ sus `service/`) | **A** |

### Protocolo de `db/sqlc` (bomba de relojería)

`make generate` **reescribe toda la carpeta `db/sqlc/`**. Si los dos lo corren a la vez, se pisan todos los archivos generados.

1. Si necesitás una query nueva, **la pide el que la necesita**.
2. **A** la agrega a `db/queries/<tabla>_queries.sql` y corre `make generate`.
3. **A** commitea `db/queries/` **y** `db/sqlc/` juntos.
4. El otro hace `git pull`.

### Si sos un agente de IA

Antes de editar, identificate: **"Soy A"** o **"Soy B"**. No toques los archivos del otro ni los compartidos (`main.go`, `db/sqlc/`, `Makefile`, `docker-compose.yml`, `README.md`) salvo que te lo pidan explícitamente.

## Ramas y flujo de trabajo

- `main`: rama principal remota.
- `tp1` / `tp2`: entregas anteriores. La README indica que el flujo del TP vive en `tp2`.
- `rochatomas-ing`: rama actual de desarrollo.
- `tp3`: rama de **integración** del TP3. Las ramas de trabajo salen de acá.

Regla: no pushear a `main` directo. Los cambios sobre el TP van en la rama de trabajo correspondiente.

### Ramas del TP3

```
main            ●───●───────────●          (no tocar)
                       \
rochatomas-ing          ● 9765a8e           (desarrollo actual)
                                 \
                                  ● tp3        (integración)
                                 /    \
                            tp3-a      tp3-b    (trabajo)
```

**Las ramas no están apiladas**: cada una es una etiqueta que apunta a un commit, y ninguna "apunta" a otra. Lo único que importa es **de qué commit se parte**, y por eso `tp3-a` y `tp3-b` salen de `tp3`.

> **Guion, no barra.** `tp3-a` sí; `tp3/a` **no**, porque en `.git/refs/heads/` `tp3` es un archivo y `tp3/a` sería una carpeta con el mismo nombre: colisionan y git se rompe.

**Puesta en marcha (la hace A primero, porque `tp3` no existe):**

```bash
git checkout rochatomas-ing
git checkout -b tp3
git push -u origin tp3

git checkout -b tp3-a
git push -u origin tp3-a
```

**B, una vez que A subió `tp3`:**

```bash
git fetch origin
git checkout tp3
git checkout -b tp3-b
git push -u origin tp3-b
```

**Ciclo diario de trabajo:**

```bash
git checkout tp3-a
git fetch origin
git merge origin/tp3     # traerse lo que el otro ya integró

# ... trabajar ...

git add <solo tus archivos>     # NUNCA git add .
git commit -m "feat(courses): CRUD completo de materias"
git push origin tp3-a
```

**Mergear a `tp3`** (uno a la vez, no los dos simultáneamente):

```bash
git checkout tp3
git fetch origin && git pull origin tp3
git merge --no-ff tp3-a
git merge --no-ff tp3-b
git push origin tp3
```

### Reglas de Git que no se rompen

| Regla | Por qué |
| --- | --- |
| `git add <archivos tuyos>`, **nunca `git add .`** | `git add .` se come los archivos del otro y genera conflictos falsos |
| **Nunca** editar `db/sqlc/` a mano | Es generado; los cambios se pierden en el próximo `make generate` |
| **Nunca** `git push --force` a `tp3` | Borra el trabajo del otro |
| **Nunca** mergear dos ramas a la vez | El segundo merge ve el repo en estado raro |
| **Nunca** commitear `.env` | Está en `.gitignore`; son credenciales reales |
| `git pull` de `tp3` **siempre** antes de codear | Te ahorra el conflicto de mañana |
| Commits chicos y frecuentes | Mergear 10 commits chicos es mucho más fácil que uno de 3000 líneas |

## Secretos y seguridad

- `.env` tiene las credenciales y está en `.gitignore`. Nunca committear `.env` ni credenciales reales. Usar `.env.example` como plantilla.
- `password_hash` se guarda hasheado (bcrypt, prefijo `$2a$`); nunca loguear ni devolver hashes en respuestas.

## Checklist antes de dar por cerrada una tarea

1. ¿Cambiaste `db/queries/`? Correr `make generate` y commitear también `db/sqlc/`. **Sólo A lo hace.**
2. ¿Cambiaste `schema.sql`? Probar con `make down && make up` (el esquema solo se aplica al crear el volumen).
3. ¿Escribiste código nuevo? Correr `make test` (y `go vet .`/`go build` si tocaste Go fuera de db/sqlc).
4. ¿Los tests pasan? Confirmado antes de declarar la tarea terminada.
5. ¿Escribiste un endpoint? Comprobar punto por punto:
   - Responde **JSON**, nunca texto plano, con `Content-Type: application/json`.
   - Status correcto: `201` create · `200` read/update · `204` delete · `400` validación · `404` no existe · `405` método · `409` UNIQUE · `500` otro.
   - `sql.ErrNoRows` → `404`, **no** `500`.
   - UUID inválido en la URL → `400`, **no** panic.
   - No filtra `password_hash`.
   - No inventa endpoints del alumno que no están en [la lista](#endpoints-del-mvp).
6. ¿Probaste el endpoint con `curl` **y** lo agregaste a `requests.sh`? Probar los casos de error también, no sólo el feliz.
7. ¿Probaste dentro de Docker y no sólo con `go run`? Dentro de Docker el host de la BD es `database`, no `localhost`.
8. ¿Hiciste `git pull` de `tp3` antes de mergear, y commiteaste **sólo tus archivos**?
