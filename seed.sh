#!/usr/bin/env bash
# seed.sh — carga los datos de prueba de seed.sql en la base de Docker.
#
# Uso:
#   bash seed.sh
#
# Requisito: la base tiene que estar levantada (make up).
# Se puede correr las veces que quieras: no duplica filas y restaura lo que
# borran las pruebas (por ejemplo, el aviso 5555… que borra el caso 5).
#
# OJO: no correr esto antes de `make test`: los tests de queries esperan la
# base vacía (hacen List sobre una base recién creada).

set -euo pipefail

# Mismo nombre de proyecto que usa el Makefile (COMPOSE := docker compose -p notifyed).
cd "$(dirname "$0")"

if ! docker compose -p notifyed exec -T database \
  sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < seed.sql; then
  echo "❌ No se pudo cargar el seed. ¿La base está levantada? (make up)"
  exit 1
fi

echo "✅ Datos de prueba cargados (seed.sql)"
