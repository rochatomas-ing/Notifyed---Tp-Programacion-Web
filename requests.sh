#!/usr/bin/env bash
# requests.sh — pruebas de la API de Notifyed con curl.
#
# Uso:
#   1. Levantar la base:            make up
#   2. Levantar el server:          make run      (en otra terminal)
#   3. Cargar los datos de prueba:  bash seed.sh
#   4. Correr las pruebas:          bash requests.sh
#
# Los UUID fijos que aparecen abajo (1111…, 3333…, 4444…, 5555…) salen del seed.sql.
# El sub_token de la cursada del seed es "sub-token-de-prueba" (sección INSCRIPCIONES).
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
#  INSCRIPCIONES (B)
# ══════════════════════════════════════════════════════════════
# El alumno usa el sub_token del QR ("sub-token-de-prueba" en el seed);
# el profesor, el UUID de la cursada (3333…).

# ── Casos felices ─────────────────────────────────────────────

# ── Caso 17: POST alta con el token del QR (espero 201) ──
# Guardo el id que devuelve el alta para los casos del panel (toggle, delete).
echo "Caso 17: POST /semesters/{token}/inscriptions — alta (espero 201)"
ALTA=$(curl -s -X POST http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"alumno@test.com"}')
echo "$ALTA"
ID_INSCRIPCION=$(echo "$ALTA" | sed -n 's/.*"id_inscription":"\([^"]*\)".*/\1/p')

# ── Caso 18: GET estado de la suscripción (espero 200, active true) ──
echo "Caso 18: GET estado de la suscripción (espero 200)"
curl -i "http://localhost:8080/semesters/sub-token-de-prueba/inscriptions?email=alumno@test.com"

# ── Caso 19: DELETE baja lógica (espero 204, sin body) ──
echo "Caso 19: DELETE /semesters/{token}/inscriptions — baja (espero 204)"
curl -i -X DELETE http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"alumno@test.com"}'

# ── Caso 20: el estado después de la baja (espero 200, active false) ──
echo "Caso 20: GET estado tras la baja (espero 200, active false)"
curl -i "http://localhost:8080/semesters/sub-token-de-prueba/inscriptions?email=alumno@test.com"

# ── Caso 21: volver a escanear el QR reactiva (espero 201) ──
echo "Caso 21: POST alta de nuevo — reactivación (espero 201)"
curl -i -X POST http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"alumno@test.com"}'

# ── Caso 22: GET listado de inscriptos de la cursada (espero 200) ──
echo "Caso 22: GET /semesters/{id}/inscriptions — listado (espero 200)"
curl -i http://localhost:8080/semesters/33333333-3333-3333-3333-333333333333/inscriptions

# ── Caso 23: listado de una cursada sin inscriptos (espero 200, []) ──
echo "Caso 23: GET listado sin datos (espero 200, [])"
curl -i http://localhost:8080/semesters/99999999-9999-9999-9999-999999999999/inscriptions

# ── Caso 24: contador de suscriptos activos (espero 200, 1) ──
echo "Caso 24: GET /semesters/{id}/suscriptos (espero 200)"
curl -i http://localhost:8080/semesters/33333333-3333-3333-3333-333333333333/suscriptos

# ── Caso 25: toggle — suspender (espero 200, active false) ──
echo "Caso 25: POST /inscriptions/{id}/toggle — suspender (espero 200)"
curl -i -X POST "http://localhost:8080/inscriptions/$ID_INSCRIPCION/toggle"

# ── Caso 26: toggle de nuevo — reactivar (espero 200, active true) ──
echo "Caso 26: POST toggle de nuevo — reactivar (espero 200)"
curl -i -X POST "http://localhost:8080/inscriptions/$ID_INSCRIPCION/toggle"

# ── Caso 27: GET historial de avisos de la cursada (espero 200) ──
echo "Caso 27: GET /semesters/{id}/notices — historial (espero 200)"
curl -i http://localhost:8080/semesters/33333333-3333-3333-3333-333333333333/notices

# ── Caso 28: DELETE /inscriptions/{id} — borrado duro (espero 204) ──
# Doy de alta un segundo alumno sólo para borrarlo.
echo "Caso 28: DELETE /inscriptions/{id} — borrado duro (espero 204)"
BORRAR=$(curl -s -X POST http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"borrar@test.com"}')
ID_BORRAR=$(echo "$BORRAR" | sed -n 's/.*"id_inscription":"\([^"]*\)".*/\1/p')
curl -i -X DELETE "http://localhost:8080/inscriptions/$ID_BORRAR"

# ── Caso 29: toggle de la borrada (espero 404) ──
echo "Caso 29: POST toggle de una inscripción borrada (espero 404)"
curl -i -X POST "http://localhost:8080/inscriptions/$ID_BORRAR/toggle"

# ── Casos de error ────────────────────────────────────────────

# ── Caso 30: alta con email sin formato (espero 400) ──
echo "Caso 30: POST alta con email inválido (espero 400)"
curl -i -X POST http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"no-es-un-email"}'

# ── Caso 31: alta con email vacío (espero 400) ──
echo "Caso 31: POST alta con email vacío (espero 400)"
curl -i -X POST http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":""}'

# ── Caso 32: alta con email de más de 80 caracteres (espero 400) ──
echo "Caso 32: POST alta con email de más de 80 caracteres (espero 400)"
curl -i -X POST http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"este-email-es-demasiado-largo-para-la-validacion-de-ochenta-caracteres-y-por-eso-se-rechaza@test.com"}'

# ── Caso 33: alta con token inexistente (espero 404) ──
echo "Caso 33: POST alta con token inexistente (espero 404)"
curl -i -X POST http://localhost:8080/semesters/token-que-no-existe/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"alumno2@test.com"}'

# ── Caso 34: baja de un email que nunca se inscribió (espero 404) ──
echo "Caso 34: DELETE baja nunca inscripto (espero 404)"
curl -i -X DELETE http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"nunca@test.com"}'

# ── Caso 35: estado de un email que nunca se inscribió (espero 404) ──
echo "Caso 35: GET estado nunca inscripto (espero 404)"
curl -i "http://localhost:8080/semesters/sub-token-de-prueba/inscriptions?email=nunca@test.com"

# ── Caso 36: estado sin el parámetro email (espero 400) ──
echo "Caso 36: GET estado sin email (espero 400)"
curl -i http://localhost:8080/semesters/sub-token-de-prueba/inscriptions

# ── Caso 37: POST con JSON inválido (espero 400) ──
echo "Caso 37: POST con JSON inválido (espero 400)"
curl -i -X POST http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d 'esto no es json'

# ── Caso 38: toggle con UUID inválido en la URL (espero 400) ──
echo "Caso 38: POST toggle con UUID inválido (espero 400)"
curl -i -X POST http://localhost:8080/inscriptions/abc/toggle

# ── Caso 39: PATCH /inscriptions/{id} — método no permitido (espero 405) ──
echo "Caso 39: PATCH /inscriptions/{id} (espero 405)"
curl -i -X PATCH "http://localhost:8080/inscriptions/$ID_INSCRIPCION"

# ── Caso 40: alta a una cursada cerrada (espero 409) ──
# Se cierra la cursada del seed (end_date pasada), se prueba el alta y se
# vuelve a abrir para que el resto del archivo quede igual que antes.
echo "Caso 40a: PUT — cierro la cursada del seed (espero 200)"
curl -i -X PUT http://localhost:8080/semesters/33333333-3333-3333-3333-333333333333 \
  -H "Content-Type: application/json" \
  -d '{"end_date":"2020-01-01"}'

echo "Caso 40: POST alta a cursada cerrada (espero 409)"
curl -i -X POST http://localhost:8080/semesters/sub-token-de-prueba/inscriptions \
  -H "Content-Type: application/json" \
  -d '{"email":"tarde@test.com"}'

echo "Caso 40b: PUT — reabro la cursada (espero 200)"
curl -i -X PUT http://localhost:8080/semesters/33333333-3333-3333-3333-333333333333 \
  -H "Content-Type: application/json" \
  -d '{"end_date":"2026-12-15"}'

# ══════════════════════════════════════════════════════════════
#  USERS (A)
# ══════════════════════════════════════════════════════════════
# El body de POST es {fullname, email, password}: el id lo genera la base y el
# password se guarda hasheado (bcrypt). La respuesta nunca incluye password_hash.

# ── Casos felices ─────────────────────────────────────────────

# ── Caso 41: POST /users — crear usuario (espero 201) ──
# Guardo el id que devuelve el alta para el resto de los casos.
echo "Caso 41: POST /users — crear usuario (espero 201)"
ALTA_USER=$(curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"fullname":"Usuario de Prueba","email":"usuario@prueba.com","password":"pass123"}')
echo "$ALTA_USER"
ID_USUARIO=$(echo "$ALTA_USER" | sed -n 's/.*"id_user":"\([^"]*\)".*/\1/p')

# ── Caso 42: GET /users — listar todos (espero 200) ──
echo "Caso 42: GET /users — listar todos (espero 200)"
curl -i http://localhost:8080/users

# ── Caso 43: GET /users/{id} — obtener uno (espero 200) ──
echo "Caso 43: GET /users/{id} — obtener uno (espero 200)"
curl -i http://localhost:8080/users/$ID_USUARIO

# ── Caso 44: PUT /users/{id} — editar nombre (espero 200) ──
echo "Caso 44: PUT /users/{id} — editar nombre (espero 200)"
curl -i -X PUT http://localhost:8080/users/$ID_USUARIO \
  -H "Content-Type: application/json" \
  -d '{"fullname":"Nombre Editado"}'

# ── Caso 45: DELETE /users/{id} — borrar (espero 204, sin body) ──
echo "Caso 45: DELETE /users/{id} — borrar (espero 204, sin body)"
curl -i -X DELETE http://localhost:8080/users/$ID_USUARIO

# ── Caso 46: GET del usuario recién borrado (espero 404) ──
echo "Caso 46: GET del usuario borrado (espero 404)"
curl -i http://localhost:8080/users/$ID_USUARIO

# ── Casos de error ────────────────────────────────────────────

# ── Caso 47: POST con email inválido (espero 400) ──
echo "Caso 47: POST con email inválido (espero 400)"
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"fullname":"Usuario Inválido","email":"usuario-invalido","password":"pass123"}'

# ── Caso 48: POST con password vacío (espero 400) ──
echo "Caso 48: POST con password vacío (espero 400)"
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"fullname":"Usuario Sin Password","email":"sin-password@prueba.com","password":""}'

# ── Caso 49: GET con UUID inválido en la URL (espero 400) ──
echo "Caso 49: GET con UUID inválido en la URL (espero 400)"
curl -i http://localhost:8080/users/abc

# ── Caso 50: PUT con id inexistente (espero 404) ──
echo "Caso 50: PUT con id inexistente (espero 404)"
curl -i -X PUT http://localhost:8080/users/99999999-9999-9999-9999-999999999999 \
  -H "Content-Type: application/json" \
  -d '{"fullname":"Este usuario no existe"}'

# ── Caso 51: DELETE con id inexistente (espero 404) ──
echo "Caso 51: DELETE con id inexistente (espero 404)"
curl -i -X DELETE http://localhost:8080/users/99999999-9999-9999-9999-999999999999

# ── Caso 52: PATCH /users — método no permitido (espero 405) ──
echo "Caso 52: PATCH /users — método no permitido (espero 405)"
curl -i -X PATCH http://localhost:8080/users

# ── Caso 53: POST con email ya registrado (espero 409) ──
echo "Caso 53: POST con email duplicado (espero 409)"
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"fullname":"Usuario Duplicado","email":"profesor@test.com","password":"pass123"}'

# ══════════════════════════════════════════════════════════════
#  SEMESTERS (A)
# ══════════════════════════════════════════════════════════════
# POST usa la materia (2222…) y el profesor (1111…) del seed. Si no se manda
# sub_token, el service genera uno.

# ── Casos felices ─────────────────────────────────────────────

# ── Caso 54: POST /semesters — crear cursada (espero 201) ──
# Guardo el id que devuelve el alta para el resto de los casos.
echo "Caso 54: POST /semesters — crear cursada (espero 201)"
ALTA_SEM=$(curl -s -X POST http://localhost:8080/semesters \
  -H "Content-Type: application/json" \
  -d '{"id_course":"22222222-2222-2222-2222-222222222222","professor_id":"11111111-1111-1111-1111-111111111111","year":2026,"end_date":"2026-11-30"}')
echo "$ALTA_SEM"
ID_SEMESTER=$(echo "$ALTA_SEM" | sed -n 's/.*"id_semester":"\([^"]*\)".*/\1/p')

# ── Caso 55: GET /semesters?professor_id=… — listar por profesor (espero 200) ──
echo "Caso 55: GET /semesters?professor_id=… — listar por profesor (espero 200)"
curl -i "http://localhost:8080/semesters?professor_id=11111111-1111-1111-1111-111111111111"

# ── Caso 56: GET /semesters sin professor_id (espero 400) ──
echo "Caso 56: GET /semesters sin professor_id (espero 400)"
curl -i http://localhost:8080/semesters

# ── Caso 57: GET /semesters/{id} — obtener una (espero 200) ──
echo "Caso 57: GET /semesters/{id} — obtener una (espero 200)"
curl -i http://localhost:8080/semesters/$ID_SEMESTER

# ── Caso 57b: GET /semesters/{sub_token} — resolución del QR (espero 200) ──
echo "Caso 57b: GET /semesters/{sub_token} — resolución del QR (espero 200)"
curl -i http://localhost:8080/semesters/sub-token-de-prueba

# ── Caso 57c: GET /profesores/{id}/semesters (espero 200) ──
echo "Caso 57c: GET /profesores/{id}/semesters (espero 200)"
curl -i http://localhost:8080/profesores/11111111-1111-1111-1111-111111111111/semesters

# ── Caso 58: PUT /semesters/{id} — corregir fecha de cierre (espero 200) ──
echo "Caso 58: PUT /semesters/{id} — corregir fecha de cierre (espero 200)"
curl -i -X PUT http://localhost:8080/semesters/$ID_SEMESTER \
  -H "Content-Type: application/json" \
  -d '{"end_date":"2026-12-01"}'

# ── Caso 59: DELETE /semesters/{id} — borrar (espero 204, sin body) ──
echo "Caso 59: DELETE /semesters/{id} — borrar (espero 204, sin body)"
curl -i -X DELETE http://localhost:8080/semesters/$ID_SEMESTER

# ── Caso 60: GET de la cursada recién borrada (espero 404) ──
echo "Caso 60: GET de la cursada borrada (espero 404)"
curl -i http://localhost:8080/semesters/$ID_SEMESTER

# ── Casos de error ────────────────────────────────────────────

# ── Caso 61: POST con fecha de cierre inválida (espero 400) ──
echo "Caso 61: POST con fecha de cierre inválida (espero 400)"
curl -i -X POST http://localhost:8080/semesters \
  -H "Content-Type: application/json" \
  -d '{"id_course":"22222222-2222-2222-2222-222222222222","professor_id":"11111111-1111-1111-1111-111111111111","year":2026,"end_date":"13-8-2026"}'

# ── Caso 62: POST con campos obligatorios vacíos (espero 400) ──
echo "Caso 62: POST con campos obligatorios vacíos (espero 400)"
curl -i -X POST http://localhost:8080/semesters \
  -H "Content-Type: application/json" \
  -d '{"id_course":"","professor_id":"","year":0,"end_date":""}'

# ── Caso 63: GET con identificador inexistente (espero 404) ──
# "abc" no es UUID, así que la ruta lo trata como sub_token: no existe → 404.
echo "Caso 63: GET /semesters/abc — token inexistente (espero 404)"
curl -i http://localhost:8080/semesters/abc

# ── Caso 64: GET con professor_id inválido (espero 400) ──
echo "Caso 64: GET con professor_id inválido (espero 400)"
curl -i "http://localhost:8080/semesters?professor_id=no-es-un-uuid"

# ── Caso 65: PUT con id inexistente (espero 404) ──
echo "Caso 65: PUT con id inexistente (espero 404)"
curl -i -X PUT http://localhost:8080/semesters/99999999-9999-9999-9999-999999999999 \
  -H "Content-Type: application/json" \
  -d '{"end_date":"2027-06-30"}'

# ── Caso 66: PUT con fecha de cierre inválida (espero 400) ──
echo "Caso 66: PUT con fecha de cierre inválida (espero 400)"
curl -i -X PUT http://localhost:8080/semesters/33333333-3333-3333-3333-333333333333 \
  -H "Content-Type: application/json" \
  -d '{"end_date":"13-30-2027"}'

# ── Caso 67: DELETE con id inexistente (espero 404) ──
echo "Caso 67: DELETE con id inexistente (espero 404)"
curl -i -X DELETE http://localhost:8080/semesters/99999999-9999-9999-9999-999999999999

# ── Caso 68: PATCH /semesters — método no permitido (espero 405) ──
echo "Caso 68: PATCH /semesters — método no permitido (espero 405)"
curl -i -X PATCH http://localhost:8080/semesters

# ══════════════════════════════════════════════════════════════
#  COURSES (A)
# ══════════════════════════════════════════════════════════════
# POST sólo lleva el nombre: el id lo genera la base.

# ── Casos felices ─────────────────────────────────────────────

# ── Caso 69: POST /courses — crear materia (espero 201) ──
# Guardo el id que devuelve el alta para el resto de los casos.
echo "Caso 69: POST /courses — crear materia (espero 201)"
ALTA_CURSO=$(curl -s -X POST http://localhost:8080/courses \
  -H "Content-Type: application/json" \
  -d '{"name":"Curso de Prueba"}')
echo "$ALTA_CURSO"
ID_CURSO=$(echo "$ALTA_CURSO" | sed -n 's/.*"id_course":"\([^"]*\)".*/\1/p')

# ── Caso 70: GET /courses — listar todas (espero 200) ──
echo "Caso 70: GET /courses — listar todas (espero 200)"
curl -i http://localhost:8080/courses

# ── Caso 71: GET /courses/{id} — obtener una (espero 200) ──
echo "Caso 71: GET /courses/{id} — obtener una (espero 200)"
curl -i http://localhost:8080/courses/$ID_CURSO

# ── Caso 71b: GET /courses/{id}/semesters — cursadas de la materia (espero 200) ──
echo "Caso 71b: GET /courses/{id}/semesters (espero 200)"
curl -i http://localhost:8080/courses/22222222-2222-2222-2222-222222222222/semesters

# ── Caso 72: PUT /courses/{id} — editar nombre (espero 200) ──
echo "Caso 72: PUT /courses/{id} — editar nombre (espero 200)"
curl -i -X PUT http://localhost:8080/courses/$ID_CURSO \
  -H "Content-Type: application/json" \
  -d '{"name":"Nombre Editado del Curso"}'

# ── Caso 73: DELETE /courses/{id} — borrar (espero 204, sin body) ──
echo "Caso 73: DELETE /courses/{id} — borrar (espero 204, sin body)"
curl -i -X DELETE http://localhost:8080/courses/$ID_CURSO

# ── Caso 74: GET de la materia recién borrada (espero 404) ──
echo "Caso 74: GET de la materia borrada (espero 404)"
curl -i http://localhost:8080/courses/$ID_CURSO

# ── Casos de error ────────────────────────────────────────────

# ── Caso 75: PATCH /courses — método no permitido (espero 405) ──
echo "Caso 75: PATCH /courses — método no permitido (espero 405)"
curl -i -X PATCH http://localhost:8080/courses

# ── Caso 76: POST con nombre vacío (espero 400) ──
echo "Caso 76: POST con nombre vacío (espero 400)"
curl -i -X POST http://localhost:8080/courses \
  -H "Content-Type: application/json" \
  -d '{"name":""}'

# ── Caso 77: POST con nombre de más de 40 caracteres (espero 400) ──
echo "Caso 77: POST con nombre de más de 40 caracteres (espero 400)"
curl -i -X POST http://localhost:8080/courses \
  -H "Content-Type: application/json" \
  -d '{"name":"Este nombre de curso es demasiado largo y excede el límite permitido de caracteres"}'

# ── Caso 78: GET con UUID inválido en la URL (espero 400) ──
echo "Caso 78: GET con UUID inválido en la URL (espero 400)"
curl -i http://localhost:8080/courses/abc

# ── Caso 79: PUT con id inexistente (espero 404) ──
echo "Caso 79: PUT con id inexistente (espero 404)"
curl -i -X PUT http://localhost:8080/courses/99999999-9999-9999-9999-999999999999 \
  -H "Content-Type: application/json" \
  -d '{"name":"Esta materia no existe"}'

# ── Caso 80: DELETE con id inexistente (espero 404) ──
echo "Caso 80: DELETE con id inexistente (espero 404)"
curl -i -X DELETE http://localhost:8080/courses/99999999-9999-9999-9999-999999999999

# EOF
