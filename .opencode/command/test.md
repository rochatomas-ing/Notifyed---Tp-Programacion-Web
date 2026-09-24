---
description: Corre el flujo de tests del repo (levanta la base si hace falta y ejecuta make test).
---

Corré el flujo de tests del proyecto tal como lo documenta AGENTS.md y el skill notifyed-tooling:
1. Confirmá que exista `.env` (si no, copialo de `.env.example`).
2. Levantá la base con `make up` (si el contenedor no está arriba).
3. Corré `make test`.
4. Informá el resultado: si algo falla, mostrá el error y proponé el diagnóstico, sin arreglar nada todavía salvo que el usuario lo pida.