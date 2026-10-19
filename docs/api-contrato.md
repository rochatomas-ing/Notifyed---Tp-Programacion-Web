# Contrato de la API — Notifyed (TP3)

Acuerdos entre A y B sobre rutas, formas JSON y status codes. Se escribe
**antes** de codear la Fase 3 y cualquier cambio se avisa antes de tocar
código. Las decisiones marcadas ⚠️ son propuestas y necesitan el ok de los dos.

## Reglas generales

| Regla | Detalle |
| --- | --- |
| Formato | Siempre JSON, también los errores. `Content-Type: application/json` |
| Formato de error | `{"error": "mensaje en español"}` |
| Status | 201 crear · 200 leer/actualizar · 204 borrar · 400 validación · 404 no existe · 405 método · 409 conflicto · 500 error interno (+ log) |
| `sql.ErrNoRows` | → **404**, nunca 500 |
| UUID inválido (URL o body) | → **400**, nunca panic |
| `password_hash` | Nunca sale: POST /users responde `UserDTO` |
| Idioma | Rutas en inglés, mensajes en español |
| Autenticación | Ninguna en este TP |

## Decisiones a validar (⚠️)

| # | Tema | Propuesta |
| --- | --- | --- |
| 1 | Alta a una cursada cerrada (`end_date < hoy`) | `409 {"error":"la cursada está cerrada"}` |
| 2 | `GET` estado de suscripción, email que nunca se inscribió | `404 {"error":"no hay inscripción con ese email"}` |
| 3 | Baja de una inscripción que ya estaba inactiva | `204` (idempotente: la fila existe) |
| 4 | `GET /semesters/{ref}/inscriptions` con `ref` UUID vs token | UUID → listado (profesor). Token → estado del alumno, `?email=` obligatorio (400 si falta) |
| 5 | `GET /semesters/{id}/suscriptos` | `{"active_subscribers": 34}` (objeto, no número suelto) |
| 6 | Validación de email | Obligatorio, ≤ 80 caracteres (varchar(80)), formato validado con `net/mail` |
| 7 | `PUT /notices/{id}` | Solo `message`, no `title` (ya implementado así) |
| 8 | `GET /semesters/{sub_token}` | El ejemplo del roadmap trae `active_subscribers`; la query actual no lo devuelve. Propuesta: no agregarlo (el panel del profesor ya lo tiene en `/profesores/{id}/semesters`) y corregir el ejemplo. Si se necesita en el QR, A agrega la query |

---

# Parte del ALUMNO (B)

El alumno **no tiene cuenta**: se identifica con su email. `{sub_token}` es el
token del QR de la cursada (`semesters.sub_token`). La UNIQUE `(id_semester,
email)` garantiza una sola fila por alumno y cursada: la baja es **lógica**
(`active = false`) y volver a escanear el QR **reactiva** en lugar de duplicar.

## POST /semesters/{sub_token}/inscriptions · alta · → 201

Body:

```json
{ "email": "alumno@unc.edu.ar" }
```

Respuesta `201`:

```json
{
  "id_inscription": "uuid",
  "id_semester": "uuid",
  "email": "alumno@unc.edu.ar",
  "active": true,
  "created_at": "2026-10-18T10:00:00Z"
}
```

- Si el email ya estaba inscripto, **no** da 409: `ON CONFLICT DO UPDATE SET
  active = true` lo reactiva y responde 201 igual.
- Token inexistente → `404`. Email inválido → `400`. Cursada cerrada → ⚠️ 409.

Queries: `GetSemesterByToken` + `CreateInscription`.

## DELETE /semesters/{sub_token}/inscriptions · baja · → 204 | 404

Body:

```json
{ "email": "alumno@unc.edu.ar" }
```

- Fila existente (activa o no) → `204` sin body.
- Nunca inscripto → `404 {"error":"no hay inscripción con ese email"}`.
- Token inexistente → `404`.

Queries: `GetSemesterByToken` + `SetInscriptionActive(false)`.

## GET /semesters/{sub_token}/inscriptions?email=… · estado · → 200 | 404

El frontend del link del QR resuelve qué mostrar con esto.

Respuesta `200`: la inscripción completa (mismo JSON que el alta).
Nunca inscripto → `404`. Sin `email` → `400`. Token inexistente → `404`.

Queries: `GetSemesterByToken` + `GetInscriptionByEmailAndSemester`.

---

# Parte del PROFESOR (B)

## GET /semesters/{id}/inscriptions · quién está anotado · → 200

Con `{id}` UUID devuelve el listado completo (activos e inactivos), ordenado
por `created_at`:

```json
[
  {
    "id_inscription": "uuid",
    "id_semester": "uuid",
    "email": "alumno@unc.edu.ar",
    "active": true,
    "created_at": "2026-10-18T10:00:00Z"
  }
]
```

Sin datos → `200 []` (nunca `null`). Query: `ListInscriptionsBySemester`.

> ⚠️ Decisión 4: la misma ruta con `sub_token` (no UUID) atiende al alumno.
> Ruta registrada una sola vez como `/semesters/{idOrToken}/inscriptions`.

## POST /inscriptions/{id}/toggle · suspender/reactivar · → 200 | 404

Invierte `active` y devuelve la inscripción actualizada (mismo JSON que arriba).
`{id}` inexistente → `404`.

Queries: `GetInscriptionById` + `SetInscriptionActive(!active)`.
Nota: `SetInscriptionActive` filtra por `(id_semester, email)`, por eso primero
se busca la fila por id. No hace falta query nueva.

## DELETE /inscriptions/{id} · borrado duro · → 204 | 404

Borra la fila (a diferencia de la baja del alumno, que es lógica).
Inexistente → `404`. Query: `DeleteInscription`.

## GET /semesters/{id}/suscriptos · contador · → 200

```json
{ "active_subscribers": 34 }
```

Cuenta solo inscripciones con `active = true`. Query: `CountActiveBySemester`.

## GET /semesters/{id}/notices · historial publicado · → 200

Avisos de la cursada con el nombre del profesor que los publicó, ordenados
por fecha:

```json
[
  {
    "id_notice": "uuid",
    "id_semester": "uuid",
    "title": "Parcial",
    "message": "Se toma el lunes",
    "created_at": "2026-10-18T10:00:00Z",
    "sender_name": "Profesor de Prueba"
  }
]
```

Sin datos → `200 []`. `{id}` no UUID → `400`. Query: `ListNoticesWithSender`.

---

# Parte del PROFESOR (A)

Ya implementado y andando: CRUD de `courses`, `semesters` (crear, listar por
profesor, `GET /semesters/{sub_token}` con `is_open`, `PUT` para cerrar con
`end_date`, `DELETE`) y CRUD de `notices` (B, Fase 1-2).

### GET /semesters/{sub_token} · resolución del QR · → 200 | 404

Respuesta actual (implementada):

```json
{
  "id_semester": "uuid",
  "id_course": "uuid",
  "professor_id": "uuid",
  "year": 2026,
  "sub_token": "abc123",
  "end_date": "2026-12-15T00:00:00Z",
  "course_name": "Programación Web",
  "is_open": true
}
```

⚠️ Diferencias con el ejemplo del roadmap (§6): el ejemplo incluía
`"active_subscribers": 34` (ver decisión #8) y mostraba `end_date` como
`"2026-12-15"`; la implementación devuelve el timestamp completo.

Pendiente de A (Fase 3) — borrador para el ok:

## POST /users · crear cuenta de profesor · → 201

Body:

```json
{ "fullname": "Prof. Gómez", "email": "gomez@unc.edu.ar", "password": "secreta" }
```

Respuesta `201` (`UserDTO`, **sin** `password_hash`):

```json
{ "id_user": "uuid", "fullname": "Prof. Gómez", "email": "gomez@unc.edu.ar" }
```

Email duplicado (UNIQUE) → `409`. Campos vacíos o email inválido → `400`.
Query: `CreateUser` (password hasheada con bcrypt en el service).

---

# Referencia rápida de lo ya implementado

| Endpoint | Status |
| --- | --- |
| `GET /health` | 200 · 503 si la base no responde |
| `GET/POST /courses` · `GET/PUT/DELETE /courses/{id}` | 200/201/204, 400/404/405 |
| `GET /courses/{id}/semesters` | 200 |
| `POST /semesters` · `GET /semesters/{idOrToken}` | 201 / 200, 404 |
| `GET /profesores/{id}/semesters` | 200 |
| `PUT /semesters/{id}` (cerrar) · `DELETE /semesters/{id}` | 200 · 204 |
| `GET/POST /notices` · `GET/PUT/DELETE /notices/{id}` | 200/201/204, 400/404/405 |
