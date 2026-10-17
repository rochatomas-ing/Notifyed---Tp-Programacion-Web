-include .env

APP_NAME := notifyed
DB_HOST ?= localhost
DB_PORT ?= $(if $(POSTGRES_PORT),$(POSTGRES_PORT),5432)
DB_USER ?= $(POSTGRES_USER)
DB_PASSWORD ?= $(POSTGRES_PASSWORD)
DB_NAME ?= $(POSTGRES_DB)
DB_SSLMODE ?= disable
export DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME DB_SSLMODE
export TEST_DB_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
COMPOSE := docker compose -p notifyed

.PHONY: all build down up up-all run test clean

all:
	@$(MAKE) --no-print-directory build
	@$(MAKE) --no-print-directory down
	@set -e; \
	trap '$(MAKE) --no-print-directory down' EXIT; \
	$(MAKE) --no-print-directory up; \
	$(MAKE) --no-print-directory test; \
	$(MAKE) --no-print-directory clean

generate:
	@echo "Generando código con sqlc (Si ya existe es regenerado)..."
	@sqlc generate

up:
	@echo "Iniciando contenedores de Docker..."
	@$(COMPOSE) up -d --wait database

up-all:
	@echo "Iniciando la aplicación y la base de datos..."
	@$(COMPOSE) up -d --build --wait

run:
	@go run .

down:
	@echo "Deteniendo contenedores de Docker y borrando volumenes..."
	@$(COMPOSE) down -v	

build: generate
	@mkdir -p tmp
	@go build -o tmp/$(APP_NAME) .

test:
	@echo "Corriendo los tests..."
	@go test -v ./db/sqlc

clean:
	@echo "Limpiando archivos generados..."
	@rm -f tmp/$(APP_NAME)
	@rm -rf tmp	