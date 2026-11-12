#!/usr/bin/env bash
# run-app.sh — Ejecuta el flujo completo de la aplicación para el TP5:
# 1) Tareas previas: generación (sqlc + templ), compilación, limpieza previa, inicio y espera de la BD.
# 2) Ejecución del servidor web SSR en http://localhost:8080.
# 3) Tareas posteriores: baja contenedores y borra volúmenes al finalizar (Ctrl+C).

set -euo pipefail

cd "$(dirname "$0")"
exec make run-app
