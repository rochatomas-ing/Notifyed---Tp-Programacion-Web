#!/usr/bin/env bash
# requests.sh — pruebas de la API de Notifyed con curl.
#
# Uso:
#   1. Levantar la base:            make up
#   2. Levantar el server:          go run .        (en otra terminal)
#   3. Cargar los datos de prueba:  bash seed.sh
#   4. Correr las pruebas:          bash requests.sh
#
# Los UUID fijos que aparecen abajo (1111…, 3333…, 4444…, 5555…) salen del seed.sql.
# Para volver a correr todo desde cero: bash seed.sh otra vez (restaura el aviso
# que borra el caso 5).

# ══════════════════════════════════════════════════════════════
#  NOTICES (B)
# ══════════════════════════════════════════════════════════════

# ── Casos felices ─────────────────────────────────────────────

# ── Caso 1: POST /notices — crear aviso (espero 201) ──
echo "Caso 1: POST /notices — crear aviso (espero 201)"
curl -i -X POST http://localhost:8080/notices \
  -H "Content-Type: application/json" \
  -d '{"id_semester":"33333333-3333-3333-3333-333333333333","id_sender":"11111111-1111-1111-1111-111111111111","title":"Aviso de prueba","message":"Se toma parcial el lunes"}'

# ── Caso 2: GET /notices — listar todos (espero 200) ──
echo "Caso 2: GET /notices — listar (espero 200)"
curl -i http://localhost:8080/notices

# ── Caso 3: GET /notices/{id} — obtener uno (espero 200) ──
echo "Caso 3: GET /notices/{id} (espero 200)"
curl -i http://localhost:8080/notices/44444444-4444-4444-4444-444444444444

# ── Caso 4: PUT /notices/{id} — editar el mensaje (espero 200) ──
echo "Caso 4: PUT /notices/{id} — editar mensaje (espero 200)"
curl -i -X PUT http://localhost:8080/notices/44444444-4444-4444-4444-444444444444 \
  -H "Content-Type: application/json" \
  -d '{"message":"Mensaje editado con PUT"}'

# ── Caso 5: DELETE /notices/{id} — borrar (espero 204, sin body) ──
echo "Caso 5: DELETE /notices/{id} (espero 204, sin body)"
curl -i -X DELETE http://localhost:8080/notices/55555555-5555-5555-5555-555555555555

# ── Caso 6: GET del aviso recién borrado (espero 404) ──
echo "Caso 6: GET del aviso borrado (espero 404)"
curl -i http://localhost:8080/notices/55555555-5555-5555-5555-555555555555

# ── Casos de error ────────────────────────────────────────────

# ── Caso 7: POST con title vacío (espero 400) ──
echo "Caso 7: POST con title vacío (espero 400)"
curl -i -X POST http://localhost:8080/notices \
  -H "Content-Type: application/json" \
  -d '{"id_semester":"33333333-3333-3333-3333-333333333333","id_sender":"11111111-1111-1111-1111-111111111111","title":"","message":"Sin título"}'

# ── Caso 8: POST con message vacío (espero 400) ──
echo "Caso 8: POST con message vacío (espero 400)"
curl -i -X POST http://localhost:8080/notices \
  -H "Content-Type: application/json" \
  -d '{"id_semester":"33333333-3333-3333-3333-333333333333","id_sender":"11111111-1111-1111-1111-111111111111","title":"Sin mensaje","message":""}'

# ── Caso 9: POST con JSON inválido (espero 400) ──
echo "Caso 9: POST con JSON inválido (espero 400)"
curl -i -X POST http://localhost:8080/notices \
  -H "Content-Type: application/json" \
  -d 'esto no es json'

# ── Caso 10: POST con UUID inválido en el body (espero 400) ──
echo "Caso 10: POST con UUID inválido (espero 400)"
curl -i -X POST http://localhost:8080/notices \
  -H "Content-Type: application/json" \
  -d '{"id_semester":"no-soy-un-uuid","id_sender":"11111111-1111-1111-1111-111111111111","title":"Aviso","message":"Mensaje"}'

# ── Caso 11: GET con UUID inválido en la URL (espero 400) ──
echo "Caso 11: GET con UUID inválido en la URL (espero 400)"
curl -i http://localhost:8080/notices/abc

# ── Caso 12: PUT con message vacío (espero 400) ──
echo "Caso 12: PUT con message vacío (espero 400)"
curl -i -X PUT http://localhost:8080/notices/44444444-4444-4444-4444-444444444444 \
  -H "Content-Type: application/json" \
  -d '{"message":""}'

# ── Caso 13: PUT con id inexistente (espero 404) ──
echo "Caso 13: PUT con id inexistente (espero 404)"
curl -i -X PUT http://localhost:8080/notices/99999999-9999-9999-9999-999999999999 \
  -H "Content-Type: application/json" \
  -d '{"message":"Este aviso no existe"}'

# ── Caso 14: DELETE con id inexistente (espero 404) ──
echo "Caso 14: DELETE con id inexistente (espero 404)"
curl -i -X DELETE http://localhost:8080/notices/99999999-9999-9999-9999-999999999999

# ── Caso 15: PATCH /notices — método no permitido (espero 405) ──
echo "Caso 15: PATCH /notices — método no permitido (espero 405)"
curl -i -X PATCH http://localhost:8080/notices

# ── Caso 16: POST con id_sender inexistente (espero 400) ──
# (requiere que el handler valide que el profesor exista antes de insertar)
echo "Caso 16: POST con id_sender inexistente (espero 400)"
curl -i -X POST http://localhost:8080/notices \
  -H "Content-Type: application/json" \
  -d '{"id_semester":"33333333-3333-3333-3333-333333333333","id_sender":"00000000-0000-0000-0000-000000000000","title":"Aviso","message":"De un profesor que no existe"}'

# ══════════════════════════════════════════════════════════════
#  (Acá abajo Mati agrega su sección: users, courses, semesters)
# ══════════════════════════════════════════════════════════════
