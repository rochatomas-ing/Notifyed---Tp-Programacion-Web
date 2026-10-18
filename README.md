# Notifyed

Sistema de avisos para cursadas universitarias: el profesor crea materias y cursadas y publica avisos; los alumnos se suscriben por email a través de un link/QR. El repositorio incluye el backend en Go (API REST sobre PostgreSQL) y el frontend estático.

## Requisitos

- Docker y Docker Compose
- Go 1.22 o superior
- sqlc (lo usa `make` para regenerar el código de `db/sqlc/`)
- curl (para ejecutar `requests.sh`)

## Puesta en marcha

### 1. Clonar el repositorio y pararse en la rama de entrega

```bash
git clone https://github.com/rochatomas-ing/Notifyed---Tp-Programacion-Web.git
cd Notifyed---Tp-Programacion-Web
git checkout tp3
```

### 2. Crear el archivo de credenciales

```bash
cp .env.example .env
```

| Variable | Ejemplo | Descripción |
| --- | --- | --- |
| `APP_PORT` | `8080` | Puerto del servidor web |
| `POSTGRES_PORT` | `5432` | Puerto de PostgreSQL |
| `POSTGRES_USER` | `postgres` | Usuario de la base |
| `POSTGRES_PASSWORD` | `postgres` | Contraseña de la base |
| `POSTGRES_DB` | `avisos` | Nombre de la base |

## Correr los tests de queries

La base corre en Docker y los tests (`go test ./db/sqlc`) se conectan a ella usando el `.env`.

```bash
# Flujo completo: build (con generate) → down → up → test → down → clean
make

# O paso a paso:
make up      # levanta PostgreSQL y espera a que esté saludable
make test    # corre los tests de queries
```

> **Importante:** los tests esperan la base vacía. Si ya cargaste los datos de prueba (ver más abajo), recreá la base antes de testear: `make down && make up && make test`.

## Levantar el servidor

```bash
make up      # base de datos en Docker
make run     # servidor en http://localhost:8080
```

> `make run` (y no `go run .`) porque el Makefile es el que exporta las variables de conexión (`DB_HOST`, `DB_USER`...) a partir del `.env`.

## Datos de prueba y pruebas de la API

```bash
# Con la base arriba (make up):
bash seed.sh       # carga los datos de prueba (usuario, cursada y avisos)
bash requests.sh   # ejecuta las peticiones de prueba con curl
```

`seed.sh` se puede correr las veces que quieras: no duplica datos y restaura lo que borran las pruebas, así `requests.sh` es re-ejecutable.

Flujo recomendado de punta a punta:

```bash
make up
bash seed.sh
make run            # en otra terminal
bash requests.sh
```

## Comandos del Makefile

| Comando | Qué hace |
| --- | --- |
| `make` | Flujo completo: build (con generate) → down → up → test → down → clean |
| `make up` | Levanta la base de datos en Docker |
| `make down` | Baja los contenedores y borra los volúmenes (**borra los datos**) |
| `make test` | Corre los tests de queries |
| `make generate` | Regenera `db/sqlc/` con sqlc |
| `make clean` | Borra el binario temporal |

## Documentación

- [Documentación de la base de datos](https://docs.google.com/document/d/14D9cFFU1qXDRc5G7Rvl9kUP221paD0Ow21wZxIRfTSE/edit?usp=sharing)
- `docs/` — consigna del TP3 y material de la cursada
- `AGENTS.md` — contexto técnico del proyecto
