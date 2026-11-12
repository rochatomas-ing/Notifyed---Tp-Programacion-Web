-include .env

export PATH := $(shell go env GOPATH 2>/dev/null)/bin:$(HOME)/go/bin:$(PATH)

APP_NAME := notifyed
APP_PORT ?= 8080
POSTGRES_PORT ?= 5432
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB ?= avisos

DB_HOST ?= localhost
DB_PORT ?= $(POSTGRES_PORT)
DB_USER ?= $(POSTGRES_USER)
DB_PASSWORD ?= $(POSTGRES_PASSWORD)
DB_NAME ?= $(POSTGRES_DB)
DB_SSLMODE ?= disable

export APP_PORT POSTGRES_PORT POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB
export DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME DB_SSLMODE
export TEST_DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

COMPOSE := docker compose -p notifyed
SQLC := $(shell command -v sqlc 2>/dev/null || echo "go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest")
TEMPL := $(shell command -v templ 2>/dev/null || echo "go run github.com/a-h/templ/cmd/templ@v0.3.1100")

.PHONY: all run-app env generate build up up-all run down test clean

all: run-app

# Flujo completo exigido para la entrega del TP5:
# 1) Tareas previas: asegurar .env, ejecutar sqlc y templ, compilar, borrar contenedores y volúmenes previos, levantar base de datos y esperar a que esté saludable.
# 2) Ejecutar la aplicación.
# 3) Tareas posteriores (al finalizar o presionar Ctrl+C): borrar contenedores, volúmenes y archivos compilados.
run-app: env
	@echo "=== [1/3] Tareas previas: generación de código, compilación y preparación de base de datos ==="
	@$(MAKE) --no-print-directory build
	@$(MAKE) --no-print-directory down
	@set -e; \
	SERVER_PID=""; \
	cleanup() { \
		trap - EXIT INT TERM; \
		echo ""; \
		if [ -n "$$SERVER_PID" ] && kill -0 "$$SERVER_PID" 2>/dev/null; then \
			kill -TERM "$$SERVER_PID" 2>/dev/null || true; \
			wait "$$SERVER_PID" 2>/dev/null || true; \
		fi; \
		echo "=== [3/3] Tareas posteriores: borrando contenedores, volúmenes y archivos temporales ==="; \
		$(COMPOSE) down -v; \
		rm -rf tmp; \
	}; \
	trap cleanup EXIT; \
	trap 'exit 0' INT TERM; \
	$(MAKE) --no-print-directory up; \
	echo "=== [2/3] Ejecutando la aplicación en http://localhost:$(APP_PORT) (Presione Ctrl+C para detener) ==="; \
	./tmp/$(APP_NAME) & \
	SERVER_PID=$$!; \
	wait "$$SERVER_PID"

env:
	@if [ ! -f .env ]; then \
		echo "Creando archivo .env a partir de .env.example..."; \
		cp .env.example .env; \
	fi

generate:
	@echo "Generando código con sqlc..."
	@$(SQLC) generate
	@echo "Generando vistas con templ..."
	@$(TEMPL) generate

build: generate
	@echo "Compilando la aplicación Go..."
	@mkdir -p tmp
	@go build -o tmp/$(APP_NAME) .

up: env
	@echo "Iniciando contenedor de base de datos y esperando a que esté corriendo..."
	@$(COMPOSE) up -d --wait database

up-all: env generate
	@echo "Iniciando la aplicación y la base de datos en Docker..."
	@$(COMPOSE) up -d --build --wait

run: env up generate
	@echo "Iniciando el servidor Go en http://localhost:$(APP_PORT)..."
	@go run .

down:
	@echo "Deteniendo contenedores de Docker y borrando volúmenes..."
	@$(COMPOSE) down -v

test: env
	@echo "Corriendo los tests de sqlc..."
	@go test -v ./db/sqlc

clean:
	@echo "Limpiando archivos generados..."
	@rm -f tmp/$(APP_NAME)
	@rm -rf tmp
