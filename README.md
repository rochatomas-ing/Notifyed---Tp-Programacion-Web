# NotifyEd

Sistema de avisos urgentes para cursadas universitarias: el profesor administra materias y cursadas, genera códigos QR/enlaces de suscripción, gestiona alumnos inscriptos y publica avisos; los alumnos se suscriben únicamente con su correo electrónico escaneando el QR o ingresando al enlace de la cursada (sin necesidad de crear una cuenta).

Aplicación web desarrollada en **Go** (`net/http`) sobre **PostgreSQL** (`sqlc`), con **Server-Side Rendering (`templ`)**, estilos con **Pico.css** y dinamismo tipo SPA con **HTMX** (actualizaciones parciales y *Out-of-Band swaps*, manteniendo *progressive enhancement*).

> **Autores:** Rocha Tomás y Rivolta Matías — Trabajo Práctico de Programación Web.

---

## Requisitos

| Herramienta | Para qué |
| --- | --- |
| **Docker + Docker Compose** | Levanta el contenedor de PostgreSQL con el esquema inicializado automáticamente |
| **Go 1.26+** | Compilar, ejecutar el servidor web y correr la suite de tests |
| **make** | Ejecutar los comandos de orquestación definidos en el `Makefile` |
| **sqlc 1.31+** | Generar el código Go de acceso a datos desde `db/queries/` (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`) |
| **templ 0.3+** | Compilar las vistas `.templ` a Go (`go install github.com/a-h/templ/cmd/templ@latest`) |
| **curl** *(opcional)* | Ejecutar `requests.sh` si se desean correr los scripts heredados de la API REST |

> El `Makefile` incorpora automáticamente `$(HOME)/go/bin` al `PATH`, por lo que `sqlc` y `templ` instalados con `go install` funcionarán sin configuración extra.

---

## Puesta en marcha paso a paso

### 1. Clonar el repositorio y configurar variables de entorno (una sola vez)

```bash
git clone https://github.com/rochatomas-ing/Notifyed---Tp-Programacion-Web.git
cd Notifyed---Tp-Programacion-Web
cp .env.example .env
```

### 2. Iniciar la aplicación desde cero (comando recomendado)

```bash
make start
```

Este comando ejecuta el flujo completo de arranque:
1. Detiene contenedores previos y borra volúmenes (`make down`).
2. Regenera código `sqlc` / `templ` y compila la aplicación (`make build`).
3. Levanta PostgreSQL en Docker (`make up`).
4. Espera a que la base de datos esté lista para aceptar conexiones (`make wait-db`).
5. Inicia el servidor web (`make run`).

Una vez iniciado, la aplicación queda disponible en **http://localhost:8080** (`GET /health` devuelve `200 OK` cuando la conexión a PostgreSQL está activa).

### Arranque paso a paso (sin borrar datos existentes)

Si ya tenés la base levantada o querés conservar los datos cargados entre reinicios:

```bash
make up        # levanta PostgreSQL en Docker
make wait-db   # espera a que la base responda a pg_isready
make run       # regenera sqlc + templ y levanta el servidor en http://localhost:8080
```

> Usá `make run` (en lugar de `go run .` directo) porque el `Makefile` carga y exporta automáticamente las variables de conexión definidas en `.env`.

> Para detener todo y limpiar contenedores y volúmenes (borra los datos de la base):
> ```bash
> make down
> ```

---

## Configuración (`.env`)

Partiendo de la plantilla `cp .env.example .env`:

| Variable | Valor por defecto | Descripción |
| --- | --- | --- |
| `APP_PORT` | `8080` | Puerto del servidor web |
| `POSTGRES_PORT` | `5432` | Puerto expuesto de PostgreSQL |
| `POSTGRES_USER` | `postgres` | Usuario de la base de datos |
| `POSTGRES_PASSWORD` | `postgres` | Contraseña de la base de datos |
| `POSTGRES_DB` | `avisos` | Nombre de la base de datos |

---

## Ejecución de tests

Todos los tests se ejecutan contra el contenedor real de PostgreSQL (es requisito haber ejecutado `make up` y `make wait-db`).

### Suite HTTP SSR / HTMX

Valida los 10 recorridos principales de la aplicación (navegación, creación y despliegue bajo demanda de cursadas, generación y proyección de QR, portal de suscripción de alumnos, publicación in-situ con selectores dependientes Materia → Cursada, historial con filtros, bajas/toggles y swaps OOB de HTMX):

```bash
make up
make wait-db
make test-http
```

### Tests de queries `sqlc` + Suite HTTP

```bash
make down && make up && make wait-db && make test
```

> Los tests unitarios de `db/sqlc` esperan la base **vacía** (recién inicializada), por lo que conviene reiniciar el contenedor con `make down && make up` antes de correr `make test`.

### Tests heredados de API REST con `curl`

```bash
make test-api
```

Ejecuta `seed.sh` y `requests.sh` contra `http://localhost:8080`.

---

## Comandos del `Makefile`

| Comando | Qué hace |
| --- | --- |
| `make start` | Flujo completo desde cero: `down` → `build` → `up` → `wait-db` → `run` |
| `make down` | Detiene los contenedores de Docker y borra los volúmenes (**elimina los datos**) |
| `make up` | Levanta el contenedor `database` (PostgreSQL) esperando el healthcheck |
| `make wait-db` | Espera explícitamente a que PostgreSQL acepte conexiones (`pg_isready`, hasta 30 s) |
| `make generate` | Ejecuta `sqlc generate` y `templ generate` |
| `make run` | Ejecuta `generate` y levanta el servidor en `http://localhost:8080` |
| `make build` | Ejecuta `generate` y compila el binario en `tmp/notifyed` |
| `make clean` | Elimina el directorio temporal `tmp/` y el binario compilado |
| `make test-http` | Corre la suite HTTP SSR/HTMX (`http_suite_test.go`) |
| `make test` | Corre los tests de `db/sqlc` junto con la suite HTTP |
| `make test-api` | Carga `seed.sql` (`seed.sh`) y corre `requests.sh` |
| `make all` | Ciclo de verificación completo: `build` → `down` → `up` → `test` → `down` → `clean` |

---

## Apartados y rutas principales

| Sección | Rutas | Descripción |
| --- | --- | --- |
| **Materias y Cursadas** | `GET /`, `GET /materias`, `POST /courses`, `POST /courses/delete`, `POST /semesters`, `POST /semesters/delete`, `GET /courses/{id}/cursadas` | Alta y baja de materias/cursadas, despliegue bajo demanda de cursadas por materia y acceso a tokens/QR |
| **Historial de Avisos** | `GET /avisos`, `POST /avisos/delete` | Listado cronológico de avisos con filtros dinámicos por materia y cursada |
| **Gestión de Alumnos** | `GET /alumnos`, `POST /alumnos/toggle`, `POST /alumnos/delete` | Búsqueda en tiempo real por email, activación/desactivación in-situ y baja definitiva |
| **Publicar Aviso** | `GET /publicar-aviso`, `GET /publicar-aviso/cursadas`, `POST /avisos` | Selector dependiente Materia → Cursada y publicación in-situ con confirmación y reinicio del formulario |
| **QR y Suscripción de Alumnos** | `GET /qr/{token}`, `GET /cursadas/{token}/qr`, `GET /subscribir/{token}`, `POST /subscribir/{token}` | Generación de imagen PNG del QR, vista de proyección para el aula y portal de suscripción por correo |
| **Salud** | `GET /health` | Verifica conectividad con PostgreSQL (`200 OK` o `503 Service Unavailable`) |

---

## Estructura del proyecto

```text
main.go                 Entrada del servidor: conexión a BD, registro de rutas SSR/HTMX y handlers
views/                  Plantillas templ (layout, materias, avisos, alumnos, publicar aviso, QR y suscripción)
static/                 Recursos estáticos (htmx.min.js, logo.png)
db/
  schema/               Esquema SQL inicial cargado automáticamente por PostgreSQL en Docker
  queries/              Consultas SQL fuente para sqlc
  sqlc/                 Código generado por sqlc y tests de integración contra PostgreSQL
db_conn/                Conexión a PostgreSQL a partir de variables de entorno
middleware/             Middleware de logging HTTP
http_suite_test.go      Suite de tests HTTP de recorridos SSR y HTMX
Makefile                Comandos de orquestación, generación, ejecución y pruebas
docker-compose.yml      Definición del stack Docker (PostgreSQL + servicio web)
Dockerfile              Imagen multi-stage para compilar el binario en Alpine
seed.sql / seed.sh      Datos de prueba reproducibles
requests.sh             Pruebas con curl heredadas de la API REST
```

---

## Documentación

- [Documentación de la base de datos](https://docs.google.com/document/d/14D9cFFU1qXDRc5G7Rvl9kUP221paD0Ow21wZxIRfTSE/edit?usp=sharing)
