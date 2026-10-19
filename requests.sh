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
#  USERS
# ══════════════════════════════════════════════════════════════

# Casos felices

# Caso 41: POST /users — crear usuario (espero 201)
echo "Caso 41: POST /users — crear usuario (espero 201)"
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"id":"66666666-6666-6666-6666-666666666","name":"Usuario de prueba","email":"usuario@prueba.com", "password":"pass123"}'

# Caso 42: GET /users — listar todos (espero 200)
echo "Caso 42: GET /users — listar todos (espero 200)"
curl -i http://localhost:8080/users

# Caso 43: GET /users/{id} — obtener uno (espero 200)
echo "Caso 43: GET /users/{id} — obtener uno (espero 200)"
curl -i http://localhost:8080/users/66666666-6666-6666-6666-666666666

# Caso 44: PUT /users/{id} — editar nombre (espero 200)
echo "Caso 44: PUT /users/{id} — editar nombre (espero 200)"
curl -i -X PUT http://localhost:8080/users/66666666-6666-6666-6666-666666666 \
  -H "Content-Type: application/json" \
  -d '{"name":"Nombre editado"}' 

# Caso 45: DELETE /users/{id} — borrar (espero 204, sin body)
echo "Caso 45: DELETE /users/{id} — borrar (espero 204, sin body)"
curl -i -X DELETE http://localhost:8080/users/66666666-6666-6666-6666-666666666

# Caso 46: GET del usuario recién borrado (espero 404)
echo "Caso 46: GET del usuario borrado (espero 404)"
curl -i http://localhost:8080/users/66666666-6666-6666-6666-666666666

# Casos de error

#Caso 47: POST con email inválido (espero 400)
echo "Caso 47: POST con email inválido (espero 400)"
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"id":"77777777-7777-7777-7777-77777777","name":"Usuario inválido","email":"usuario-invalido","password":"pass123"}'

#Caso 48: POST con password vacío (espero 400)
echo "Caso 48: POST con password vacío (espero 400)" 
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"id":"88888888-8888-8888-8888-88888888","name":"Usuario sin password","email":"usuario@prueba.com","password":""}'

# Caso 49: GET con UUID inválido en la URL (espero 400)
echo "Caso 49: GET con UUID inválido en la URL (espero 400)"
curl -i http://localhost:8080/users/abc

# Caso 50: PUT con id inexistente (espero 404)
echo "Caso 50: PUT con id inexistente (espero 404)"
curl -i -X PUT http://localhost:8080/users/99999999-9999-9999-9999-999999999999 \
  -H "Content-Type: application/json" \
  -d '{"name":"Este usuario no existe"}'

#Caso 51: DELETE con id inexistente (espero 404)
echo "Caso 51: DELETE con id inexistente (espero 404)"
curl -i -X DELETE http://localhost:8080/users/99999999-9999-9999-9999-999999999999

#Caso 52: PATCH /users — método no permitido (espero 405)
echo "Caso 52: PATCH /users — método no permitido (espero 405)"
curl -i -X PATCH http://localhost:8080/users

#Caso 53: POST con id existente (espero 400)
echo "Caso 53: POST con id existente (espero 400)"
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"id":"11111111-1111-1111-1111-111111111111","name":"Usuario duplicado","email":"usuario@duplicado.com","password":"pass123"}'

# ══════════════════════════════════════════════════════════════
#  SEMESTERS
# ══════════════════════════════════════════════════════════════

# Casos felices

# Caso 54: POST /semesters — crear semestre (espero 201)
echo "Caso 54: POST /semesters — crear semestre (espero 201)"
curl -i -X POST http://localhost:8080/semesters \
  -H "Content-Type: application/json" \
  -d '{"id":"44444444-4444-4444-4444-444444444444", "year":"2025","name":"Semestre de prueba","end_date":"2025-06-30","course_id":"22222222-2222-2222-2222-222222222222","professor_id":"11111111-1111-1111-1111-111111111111"}'

# Caso 55: GET /semesters — listar todas las cursadas de un profesor (espero 200)
echo "Caso 55: GET /semesters — listar todas las cursadas de un profesor (espero 200)"
curl -i http://localhost:8080/semesters?professor_id=11111111-1111-1111-1111-111111111111

# Caso 56: GET /semesters — listar todos (espero 200)
echo "Caso 56: GET /semesters — listar todos (espero 200)"
curl -i http://localhost:8080/semesters

# Caso 57: GET /semesters/{id} — obtener uno (espero 200)
echo "Caso 57: GET /semesters/{id} — obtener uno (espero 200)"
curl -i http://localhost:8080/semesters/44444444-4444-4444-4444-444444444444

# Caso 58: PUT /semesters/{id} — editar fecha de finalizacion (espero 200)
echo "Caso 58: PUT /semesters/{id} — editar fecha de finalizacion (espero 200)"
curl -i -X PUT http://localhost:8080/semesters/44444444-4444-4444-4444-444444444444 \
  -H "Content-Type: application/json" \
  -d '{"end_date":"2025-09-30"}'

# Caso 59: DELETE /semesters/{id} — borrar (espero 204, sin body)
echo "Caso 59: DELETE /semesters/{id} — borrar (espero 204, sin body)"
curl -i -X DELETE http://localhost:8080/semesters/44444444-4444-4444-4444-444444444444

#Caso 60: GET del semestre recién borrado (espero 404)
echo "Caso 60: GET del semestre borrado (espero 404)"
curl -i http://localhost:8080/semesters/44444444-4444-4444-4444-444444444444

# Caso de errores

# Caso 61: POST con fecha de finaizacion inválida (espero 400)
echo "Caso 61: POST con fecha de inicio inválida (espero 400)"
curl -i -X POST http://localhost:8080/semesters \
  -H "Content-Type: application/json" \
  -d '{"id_course":"22222222-2222-2222-2222-222222222222", "professor_id":"11111111-1111-1111-1111-111111111111","name":"Semestre inválido","year":"2026","end_date":"13-8-2025"}'

# Caso 62: POST con campos obligatorios vacios (espero 400)
echo "Caso 62: POST con campos obligatorios vacios (espero 400)"
curl -i -X POST http://localhost:8080/semesters \
  -H "Content-Type: application/json" \
  -d '{"id_course":"22222222-2222-2222-2222-222222222222", "professor_id":"11111111-1111-1111-1111-111111111111","name":"","year":"","end_date":""}'

# Caso 63: GET con UUID inválido en la URL (espero 400)
echo "Caso 63: GET con UUID inválido en la URL (espero 400)"
curl -i http://localhost:8080/semesters/abc

# Caso 64: GET con UUID invalido de profesor(espero 404)
echo "Caso 64: GET con UUID invalido de profesor(espero 404)"
curl -i http://localhost:8080/semesters?professor_id=invalid_uuid

# Caso 65: PUT con id inexistente (espero 404)
echo "Caso 65: PUT con id inexistente (espero 404)"
curl -i -X PUT http://localhost:8080/semesters/99999999-9999-9999-9999-999999999999 \
  -H "Content-Type: application/json" \
  -d '{"end_date":"2027-06-30"}'

# Caso 66: PUT con formato incorrecto en fecha de finalizacion invalida (espero 400)
echo "Caso 66: PUT con fecha de finalizacion invalida (espero 400)"
curl -i -X PUT http://localhost:8080/semesters/33333333-3333-3333-3333-333333333333 \
  -H "Content-Type: application/json" \
  -d '{"end_date":"13-30-2027"}'

# Caso 67: DELETE con id inexistente (espero 404)
echo "Caso 67: DELETE con id inexistente (espero 404)"
curl -i -X DELETE http://localhost:8080/semesters/99999999-9999-9999-9999-999999999999

# Caso 68: PATCH /semesters — método no permitido (espero 405)
echo "Caso 68: PATCH /semesters — método no permitido (espero 405)"
curl -i -X PATCH http://localhost:8080/semesters

# ══════════════════════════════════════════════════════════════
#  COURSES
# ══════════════════════════════════════════════════════════════

# Casos felices

# Caso 69: POST /courses — crear curso (espero 201)
echo "Caso 69: POST /courses — crear curso (espero 201)"
curl -i -X POST http://localhost:8080/courses \
  -H "Content-Type: application/json" \
  -d '{"id_course":"77777777-7777-7777-7777-777777777","name":"Curso dde prueba"}'

# Caso 70: GET /courses — listar todos (espero 200)
echo "Caso 70: GET /courses — listar todos (espero 200)"
curl -i http://localhost:8080/courses

# Caso 71: GET /courses/{id} — obtener uno (espero 200)
echo "Caso 71: GET /courses/{id} — obtener uno (espero 200)"
curl -i http://localhost:8080/courses/77777777-7777-7777-7777-777777777

# Caso 72: PUT /courses/{id} — editar nombre (espero 200)
echo "Caso 72: PUT /courses/{id} — editar nombre (espero 200)"
curl -i -X PUT http://localhost:8080/courses/77777777-7777-7777-7777-777777777 \
  -H "Content-Type: application/json" \
  -d '{"name":"Nombre editado del curso"}'

#Caso 73: DELETE /courses/{id} — borrar (espero 204, sin body)
echo "Caso 73: DELETE /courses/{id} — borrar (espero 204, sin body)"
curl -i -X DELETE http://localhost:8080/courses/77777777-7777-7777-7777-777777777

#Caso 74: GET del curso recién borrado (espero 404)
echo "Caso 74: GET del curso borrado (espero 404)"
curl -i http://localhost:8080/courses/77777777-7777-777-7777-777777777

# Caso 75: PATCH /courses — método no permitido (espero 405)
echo "Caso 75: PATCH /courses — método no permitido (espero 405)"
curl -i -X PATCH http://localhost:8080/courses

# Casos de errores

# Caso 76: POST con campos obligatorios vacios (espero 400)
echo "Caso 76: POST con campos obligatorios vacios (espero 400)"
curl -i -X POST http://localhost:8080/courses \
  -H "Content-Type: application/json" \
  -d '{"id_course":"","name":""}'

# Caso 77: POST con campo 'name' excedido (espero 400)
echo "Caso 77: POST con campo 'name' excedido (espero 400)"
curl -i -X POST http://localhost:8080/courses \
  -H "Content-Type: application/json" \
  -d '{"id_course":"88888888-8888-8888-8888-888888888","name":"Este nombre de curso es demasiado largo y excede el límite permitido de caracteres para el campo name, lo que debería provocar un error de validación en la API"}' 

#Caso 78: GET con UUID inválido en la URL (espero 400)
echo "Caso 78: GET con UUID inválido en la URL (espero 400)"
curl -i http://localhost:8080/courses/abc

# Caso 79: PUT con id inexistente (espero 404)
echo "Caso 79: PUT con id inexistente (espero 404)"
curl -i -X PUT http://localhost:8080/courses/99999999-9999-9999-9999-999999999999 \
  -H "Content-Type: application/json" \
  -d '{"name":"Este curso no existe"}'

# Caso 80: DELETE con id inexistente (espero 404)
echo "Caso 80: DELETE con id inexistente (espero 404)"
curl -i -X DELETE http://localhost:8080/courses/99999999-9999-9999-9999-999999999999

# EOF