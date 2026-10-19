# Notifyed

Sistema de avisos para cursadas universitarias: el profesor crea materias y
cursadas y publica avisos; los alumnos se suscriben por email a través de un
link/QR (sólo alta y baja, no tienen cuenta). Backend en Go (sólo biblioteca
estándar de `net/http`) con API REST sobre PostgreSQL; frontend estático
servido por el mismo servidor.

> El detalle de rutas, cuerpos JSON y status de cada endpoint está en
> [`docs/api-contrato.md`](docs/api-contrato.md). La consigna del TP está en
> [`docs/tp3.pdf`](docs/tp3.pdf).

## Requisitos

| Herramienta | Para qué |
| --- | --- |
| Docker + Docker Compose | Camino recomendado: levanta base y servidor sin instalar nada más |
| make | Correr los atajos del paso a paso (`make up-all`, `make test`, `make down`, …) |
| curl | Ejecutar `requests.sh` (pruebas de la API) |
| Go 1.22 o superior | Sólo para correr el servidor localmente (`make run`) o los tests |
| sqlc 1.31 + | Sólo para regenerar `db/sqlc/` (`make generate`); el código generado ya está commiteado |

## Puesta en marcha paso a paso (Docker)

```bash
# 1. Clonar y preparar (una sola vez)
git clone https://github.com/rochatomas-ing/Notifyed---Tp-Programacion-Web.git
cd Notifyed---Tp-Programacion-Web
git checkout tp3
cp .env.example .env            # credenciales de desarrollo (se puede editar)

# 2. Levantar la aplicación y probarla
make up-all                     # compila y levanta la base + el servidor
make test                       # corre los tests de las queries (base vacía)
bash seed.sh                    # carga los datos de prueba
bash requests.sh                # corre los 85 casos de la API con curl
```

Queda el servidor en **http://localhost:8080** (el frontend se sirve en `/` y
`GET /health` devuelve `200` si la base responde).

**Qué hace cada comando:**

| Comando | Qué hace |
| --- | --- |
| `make up-all` | Compila la imagen del servidor (`Dockerfile`) y levanta PostgreSQL + el web server con Docker Compose, esperando a que ambos estén saludables. Equivale a `docker compose up --build -d`. La primera vez tarda un poco más por el build. |
| `make test` | Corre los tests de las queries de `db/sqlc` contra PostgreSQL real. Necesita la base **vacía** (recién creada), por eso va antes del `seed`: si ya cargaste los datos, hacé `make down` y volvé a empezar desde `make up-all`. |
| `bash seed.sh` | Carga los datos de prueba (profesor, materia, cursada y avisos). Se puede correr las veces que quieras: no duplica filas y restaura lo que borran las pruebas. |
| `bash requests.sh` | Corre los 85 casos de la API con `curl` contra `localhost:8080` (casos felices y de error). Imprime cada caso con su status esperado (`Caso N: ...`) para comparar contra la respuesta real; se puede re-ejecutar después de cada `bash seed.sh`. |

> Para bajar todo: `make down` (detiene y borra contenedores, datos y volúmenes).

## Alternativa: correr el servidor localmente con Go

```bash
make up     # levanta sólo PostgreSQL en Docker
make run    # compila y corre el servidor en http://localhost:8080
```

> `make run` (y no `go run .`) porque el Makefile exporta las variables de
> conexión (`DB_HOST`, `DB_USER`, ...) a partir del `.env`.

Para probar la API en este modo: `bash seed.sh && bash requests.sh` (en otra
terminal). Dentro de Docker el host de la base es `database` y en local es
`localhost`: el `.env` cubre los dos casos sin cambios.

## Configuración (`.env`)

Partí de la plantilla `cp .env.example .env`:

| Variable | Ejemplo | Descripción |
| --- | --- | --- |
| `APP_PORT` | `8080` | Puerto del servidor web (mapeado al 8080 del contenedor) |
| `POSTGRES_PORT` | `5432` | Puerto de PostgreSQL |
| `POSTGRES_USER` | `postgres` | Usuario de la base |
| `POSTGRES_PASSWORD` | `postgres` | Contraseña de la base |
| `POSTGRES_DB` | `avisos` | Nombre de la base |

## Endpoints (resumen)

| Recurso | Endpoints |
| --- | --- |
| Avisos (profesor) | `GET /notices` · `POST /notices` · `GET /notices/{id}` · `PUT /notices/{id}` · `DELETE /notices/{id}` |
| Inscripciones (alumno) | `POST /semesters/{sub_token}/inscriptions` · `DELETE /semesters/{sub_token}/inscriptions` · `GET /semesters/{sub_token}/inscriptions?email=…` |
| Inscripciones (profesor) | `GET /semesters/{id}/inscriptions` · `GET /semesters/{id}/suscriptos` · `POST /inscriptions/{id}/toggle` · `DELETE /inscriptions/{id}` |
| Materias | `GET /courses` · `POST /courses` · `GET /courses/{id}` · `PUT /courses/{id}` · `DELETE /courses/{id}` · `GET /courses/{id}/semesters` |
| Cursadas | `POST /semesters` · `GET /semesters/{idOrToken}` · `GET /profesores/{id}/semesters` · `PUT /semesters/{id}` · `DELETE /semesters/{id}` |
| Cuentas de profesor | `GET /users` · `POST /users` · `GET /users/{id}` · `PUT /users/{id}` · `DELETE /users/{id}` |
| Historial | `GET /semesters/{id}/notices` (avisos publicados de una cursada, con el nombre del profesor) |

Reglas generales: siempre JSON (también los errores: `{"error": "..."}`),
`password_hash` nunca sale en una respuesta, sin autenticación en este TP.

## Tests

```bash
go test ./service/      # tests unitarios de la lógica de negocio (sin base ni HTTP)
make test               # tests de queries sqlc contra PostgreSQL real
```

> Los tests de queries esperan la base **vacía**. Si ya corriste `seed.sh`,
> recreá la base antes: `make down && make up && make test`.

## Comandos del Makefile

| Comando | Qué hace |
| --- | --- |
| `make up-all` | Compila la imagen Docker y levanta base + servidor |
| `make up` | Levanta sólo PostgreSQL y espera a que esté saludable |
| `make run` | Corre el servidor en local con Go (`http://localhost:8080`) |
| `make test` | Tests de queries sqlc (requiere la base arriba y vacía) |
| `make down` | Baja los contenedores y borra los volúmenes (**borra los datos**) |
| `make generate` | Regenera `db/sqlc/` con sqlc (requiere tener sqlc instalado) |
| `make build` | `generate` + compila el binario en `tmp/` |
| `make clean` | Borra el binario temporal |
| `make` | Flujo completo: build → down → up → test → down → clean (**requiere sqlc**) |

## Estructura y arquitectura

```
main.go                 entrada: conecta la BD, arma los services y registra las rutas
handlersTP/             capa HTTP: método, URL, decodificar JSON, elegir status
service/                reglas de negocio (validaciones), testeable sin HTTP ni BD
db/sqlc/                repositorio generado por sqlc (no editar a mano)
db/queries/             SQL fuente de sqlc
dto/                    formas JSON que salen por la API (p. ej. sin password_hash)
middleware/             logging de todas las peticiones
db_conn/                conexión a PostgreSQL a partir de variables de entorno
static/                 frontend estático
docs/                   consigna del TP y contrato de la API
requests.sh             pruebas de la API con curl (85 casos)
seed.sql / seed.sh      datos de prueba reproducibles
```

Cada capa habla sólo con la de abajo: `handlersTP` → `service` → `db/sqlc`.

## La consigna del TP3, punto por punto

| Punto de la consigna | Dónde está |
| --- | --- |
| 1. Conexión a la BD + instancia del repositorio sqlc | `main.go` + `db_conn/conn.go` (`GET /health` lo verifica) |
| 2. Handlers CRUD con los métodos `Create…/List…/Get…/Update…/Delete…` | `handlersTP/` + `service/` (notices, inscriptions, users, courses, semesters) |
| 3. Pruebas de la API en un script con curl | `requests.sh` (casos felices y de error de todos los endpoints) |
| 4. Instrucciones de ejecución + Docker | este README + `Dockerfile` + `docker-compose.yml` (`make up-all`) |

## Documentación

- [`docs/api-contrato.md`](docs/api-contrato.md) — contrato completo de la API (formas JSON y decisiones acordadas)
- [`docs/tp3.pdf`](docs/tp3.pdf) — consigna del trabajo práctico
- [`docs/ROADMAPTP3_DEFINITIVO.docx`](docs/ROADMAPTP3_DEFINITIVO.docx) — plan de trabajo por fases
- [Documentación de la base de datos](https://docs.google.com/document/d/14D9cFFU1qXDRc5G7Rvl9kUP221paD0Ow21wZxIRfTSE/edit?usp=sharing)
- `AGENTS.md` — contexto técnico para agentes de IA
