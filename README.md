# NotifyEd — TP5: Renderizado del Lado del Servidor (SSR) con `templ` y `sqlc`

Sistema de gestión de materias y avisos para cursadas universitarias. En esta entrega (**TP5**), se reemplazó la arquitectura de API REST + Cliente JavaScript por una aplicación con **renderizado del lado del servidor (SSR tradicional)** en Go utilizando **[`templ`](https://templ.guide/)**, **[`sqlc`](https://sqlc.dev/)**, **PostgreSQL** y **[Pico.css](https://picocss.com/)**, aplicando navegación completa y el patrón **Post/Redirect/Get (PRG)** sin código JavaScript en el cliente.

---

## Requisitos

| Herramienta | Propósito |
| --- | --- |
| **Docker + Docker Compose** | Ejecución del motor de base de datos PostgreSQL 18 e inicialización automática del esquema (`db/schema/schema.sql`). |
| **Go 1.26+** | Compilación y ejecución del servidor web y de los generadores `sqlc` y `templ`. |
| **Make / Bash** | Automatización completa del ciclo de vida del proyecto (`make run-app` o `./run-app.sh`). |

> No es obligatorio instalar manualmente los binarios globales de `sqlc` ni `templ`: si no se encuentran en el `PATH`, el `Makefile` los ejecuta automáticamente mediante `go run`.

---

## Ejecución Rápida (Flujo Automatizado Completo)

Para clonar y probar el proyecto de punta a punta con un único comando:

```bash
git clone -b tp5 https://github.com/rochatomas-ing/Notifyed---Tp-Programacion-Web.git
cd Notifyed---Tp-Programacion-Web
make run-app
```

*(También se puede ejecutar con `./run-app.sh` o simplemente `make`).*

Una vez iniciado, abrir en el navegador: **http://localhost:8080**

### ¿Qué hace `make run-app` (y `./run-app.sh`) automáticamente?

El comando ejecuta en orden las tres etapas requeridas por la cátedra:

1. **Tareas previas:**
   - Genera automáticamente el archivo `.env` a partir de `.env.example` si no existe.
   - Ejecuta `sqlc generate` para generar el código tipado de acceso a datos en `db/sqlc/`.
   - Ejecuta `templ generate` para compilar los componentes `.templ` a código Go en `views/*_templ.go`.
   - Compila el binario de la aplicación en `tmp/notifyed`.
   - Detiene y elimina contenedores y volúmenes previos (`docker compose -p notifyed down -v`) para garantizar un entorno limpio.
   - Levanta el contenedor de PostgreSQL y espera activamente (`--wait`) hasta que el *healthcheck* (`pg_isready`) confirme que la base de datos está corriendo y con el esquema cargado.
2. **Ejecución de la aplicación:**
   - Inicia el servidor Go en **http://localhost:8080**.
   - Desde el navegador se puede visualizar la lista de materias registradas, dar de alta nuevas materias mediante el formulario HTML y eliminar materias existentes con recarga completa de página (**patrón PRG**).
3. **Tareas posteriores (al detener con `Ctrl+C`):**
   - Al interrumpir el servidor (`Ctrl+C`), un `trap` captura la señal de salida y ejecuta automáticamente `docker compose -p notifyed down -v` (borrando contenedores y volúmenes) y elimina los artefactos temporales (`tmp/`).

---

## Arquitectura SSR y Componentes

### 1. Componentes de Vista (`views/`)

El frontend del TP4 fue desacoplado de JavaScript y dividido en componentes tipados de `templ`:

- **[`views/layout.templ`](views/layout.templ):**
  - `Layout(title string)`: Define la estructura base del documento HTML5 (`<!DOCTYPE html>`, `<html>`, `<head>`, `<body>`, `<main class="container">`), incluye la hoja de estilos **Pico.css v2** y renderiza el contenido hijo mediante `{ children... }`.
  - `IndexPage(courses []db.Course)`: Componente de página principal que utiliza `@Layout(...)` y compone el formulario de creación (`@EntityForm()`) junto con el listado de materias (`@EntityList(courses)`).
- **[`views/entity_form.templ`](views/entity_form.templ):**
  - `EntityForm()`: Renderiza el formulario HTML clásico (`<form method="POST" action="/courses">`) con validación de campo requerido y longitud máxima de 40 caracteres.
- **[`views/entity_list.templ`](views/entity_list.templ):**
  - `EntityList(courses []db.Course)`: Recibe el slice de materias (`[]db.Course`) obtenido con `sqlc`. Muestra un estado vacío informativo si no hay registros (`len(courses) == 0`) o renderiza la tabla con el nombre, el UUID y un formulario clásico de baja (`<form method="POST" action="/courses/delete">`) por cada materia.

### 2. Rutas del Servidor (`main.go`)

| Método y Ruta | Descripción |
| --- | --- |
| `GET /` | Obtiene todas las materias desde PostgreSQL usando `queries.ListCourses(r.Context())` de `sqlc` y renderiza el componente `views.IndexPage(courses)` en el `http.ResponseWriter`. |
| `POST /courses` | Parsea los datos enviados por el formulario (`r.ParseForm()`), valida el nombre, inserta la nueva materia con `queries.CreateCourse(r.Context(), name)` y redirige a `/` con `http.Redirect(..., http.StatusSeeOther)` aplicando el patrón **Post/Redirect/Get (PRG)**. |
| `POST /courses/delete` | Parsea el UUID enviado por formulario, elimina la materia mediante `queries.DeleteCourse(r.Context(), id)` y redirige a `/` con `http.StatusSeeOther` (**PRG**). |
| `GET /health` | Verifica la conectividad con PostgreSQL (`conn.PingContext`) y devuelve `200 OK`. |

---

## Comandos Disponibles en el `Makefile`

| Comando | Descripción |
| --- | --- |
| `make run-app` (o `make`) | **Flujo completo de entrega:** tareas previas (`sqlc` + `templ` + compilación + recreación de BD) → ejecuta la app → tareas posteriores al salir (`down -v` + `clean`). |
| `./run-app.sh` | Script bash equivalente que invoca `make run-app`. |
| `make generate` | Ejecuta `sqlc generate` y `templ generate`. |
| `make build` | Ejecuta `generate` y compila el binario en `tmp/notifyed`. |
| `make up` | Levanta únicamente PostgreSQL en Docker y espera a que esté saludable. |
| `make run` | Levanta la base de datos (`up`), regenera código (`generate`) y ejecuta `go run .`. |
| `make up-all` | Regenera código, construye la imagen Docker de la app y levanta base + servidor en contenedores. |
| `make test` | Ejecuta los tests de integración de `db/sqlc` contra PostgreSQL. |
| `make down` | Detiene los contenedores de Docker y elimina los volúmenes (`down -v`). |
| `make clean` | Elimina el directorio `tmp/` con el binario compilado. |

---

## Estructura del Proyecto

```text
.
├── main.go                 # Punto de entrada HTTP: conexión a BD, rutas SSR y patrón PRG
├── views/                  # Componentes de vista en templ (.templ y código Go generado)
│   ├── layout.templ        # Estructura HTML base con Pico.css y componente IndexPage
│   ├── entity_form.templ   # Formulario clásico POST /courses para alta de materias
│   └── entity_list.templ   # Tabla SSR de materias registradas y baja POST /courses/delete
├── db/
│   ├── schema/schema.sql   # Esquema relacional PostgreSQL
│   ├── queries/            # Consultas SQL fuente para sqlc
│   └── sqlc/               # Código Go generado por sqlc y tests de integración
├── db_conn/conn.go         # Pool de conexión a PostgreSQL mediante variables de entorno
├── middleware/logging.go   # Middleware de registro de peticiones HTTP
├── Makefile                # Automatización del ciclo de vida completo (make run-app)
├── run-app.sh              # Script ejecutable para correr el flujo completo
├── Dockerfile              # Build multi-stage para el servidor Go
└── docker-compose.yml      # Orquestación de PostgreSQL (con healthcheck) y servidor web
```

---

## Configuración (`.env`)

Si no existe un archivo `.env`, `make run-app` lo crea automáticamente a partir de [`.env.example`](.env.example):

| Variable | Valor por defecto | Descripción |
| --- | --- | --- |
| `APP_PORT` | `8080` | Puerto HTTP en el que escucha el servidor |
| `POSTGRES_PORT` | `5432` | Puerto de PostgreSQL expuesto en localhost |
| `POSTGRES_USER` | `postgres` | Usuario de PostgreSQL |
| `POSTGRES_PASSWORD` | `postgres` | Contraseña de PostgreSQL |
| `POSTGRES_DB` | `avisos` | Nombre de la base de datos |

## Documentación Adicional

- [Documentación del diseño de la Base de Datos](https://docs.google.com/document/d/14D9cFFU1qXDRc5G7Rvl9kUP221paD0Ow21wZxIRfTSE/edit?usp=sharing)
