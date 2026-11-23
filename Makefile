-include .env

export PATH := $(HOME)/go/bin:$(PATH)

APP_NAME := notifyed
DB_HOST ?= localhost
DB_PORT ?= $(if $(POSTGRES_PORT),$(POSTGRES_PORT),5432)
DB_USER ?= $(POSTGRES_USER)
DB_PASSWORD ?= $(POSTGRES_PASSWORD)
DB_NAME ?= $(POSTGRES_DB)
DB_SSLMODE ?= disable
export DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME DB_SSLMODE
export TEST_DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
COMPOSE := docker compose -p notifyed

.PHONY: all start build clean down up wait-db generate run test test-api test-http

all:
	@$(MAKE) --no-print-directory build
	@$(MAKE) --no-print-directory down
	@set -e; \
	trap '$(MAKE) --no-print-directory down' EXIT; \
	$(MAKE) --no-print-directory up; \
	$(MAKE) --no-print-directory test; \
	$(MAKE) --no-print-directory clean

# Flujo completo para iniciar la aplicación desde cero
start:
	@$(MAKE) --no-print-directory down
	@$(MAKE) --no-print-directory build
	@$(MAKE) --no-print-directory up
	@$(MAKE) --no-print-directory wait-db
	@$(MAKE) --no-print-directory run

# Limpiar contenedores y volúmenes
down:
	@echo "Deteniendo contenedores de Docker y borrando volúmenes..."
	@$(COMPOSE) down -v

# Levantar contenedor de base de datos
up:
	@echo "Iniciando contenedor de base de datos..."
	@$(COMPOSE) up -d --wait database

# Esperar a que la base de datos esté lista para aceptar conexiones
wait-db:
	@echo "Esperando a que la base de datos esté lista..."
	@for i in $$(seq 1 30); do \
		if $(COMPOSE) exec -T database sh -c 'pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' > /dev/null 2>&1; then \
			echo "Base de datos lista."; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Error: la base de datos no respondió a tiempo. ¿Ejecutaste 'make up'?"; \
	exit 1

# Generar código con sqlc y vistas con templ
generate:
	@echo "Generando código con sqlc..."
	@sqlc generate
	@echo "Generando vistas con templ..."
	@templ generate

# Ejecutar la aplicación en local
run: generate
	@echo "Iniciando el servidor Go..."
	@go run .

build: generate
	@mkdir -p tmp
	@go build -o tmp/$(APP_NAME) .

clean:
	@echo "Limpiando archivos generados..."
	@rm -f tmp/$(APP_NAME)
	@rm -rf tmp

# Tests de queries sqlc y suite HTTP SSR
test:
	@echo "Corriendo los tests de base de datos y la suite HTTP SSR..."
	@go test -v ./db/sqlc .

# Tests de la API REST (seed + requests)
test-api:
	@echo "Cargando datos de prueba (seed.sh)..."
	@bash seed.sh
	@echo "Corriendo los tests de la API (requests.sh)..."
	@bash requests.sh

# Tests de la suite HTTP SSR/HTMX
test-http:
	@echo "Corriendo la suite HTTP de recorridos SSR..."
	@go test -v -run TestHTTPSuite .
