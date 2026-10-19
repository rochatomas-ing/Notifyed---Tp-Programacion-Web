# Contrato de la API — Notifyed (TP3)

Acuerdos entre A y B sobre rutas, formas JSON y status codes. Se escribió
**antes** de codear la Fase 3; las decisiones marcadas ⚠️ quedaron acordadas
y cualquier cambio se avisa antes de tocar código.

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

## Decisiones del contrato (⚠️ acordadas)

| # | Tema | Acuerdo |
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

> Las tres rutas se registran una sola vez como
> `/semesters/{idOrToken}/inscriptions` (decisión #4); el alumno las usa con el
> `sub_token`.

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

> Decisión #4: la misma ruta con `sub_token` (no UUID) atiende al alumno.
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

Diferencias con el ejemplo del roadmap (§6): el ejemplo incluía
`"active_subscribers": 34` (ver decisión #8) y mostraba `end_date` como
`"2026-12-15"`; la implementación devuelve el timestamp completo.

## Cuentas de profesor (`users`) — A (Fase 3, implementado)

| Endpoint | Éxito | Errores |
| --- | --- | --- |
| `POST /users` | `201` → `UserDTO` | `400` campos vacíos o email inválido · `409` email duplicado · `405` |
| `GET /users` | `200` → `[UserDTO]` (vacío `[]`) | `405` |
| `GET /users/{id}` | `200` → `UserDTO` | `404` · `400` UUID inválido · `405` |
| `PUT /users/{id}` | `200` → `UserDTO` | `400` fullname vacío · `404` · `405` |
| `DELETE /users/{id}` | `204` sin body | `404` · `400` UUID inválido · `405` |

Body de `POST /users` (la password llega en texto plano y se guarda hasheada
con bcrypt; el id lo genera la base):

```json
{ "fullname": "Prof. Gómez", "email": "gomez@unc.edu.ar", "password": "secreta" }
```

Body de `PUT /users/{id}` (sólo el nombre, igual que `UpdateUserFullname`):

```json
{ "fullname": "Prof. Gómez" }
```

`UserDTO` (única forma de salida, **nunca** incluye `password_hash`):

```json
{ "id_user": "uuid", "fullname": "Prof. Gómez", "email": "gomez@unc.edu.ar" }
```

---

# Anexo: formas JSON de los endpoints ya implementados (Fases 0-2)

Documenta los endpoints construidos antes de la Fase 3, tal como responden hoy.
Los endpoints nuevos de la Fase 3 tienen su propia sección más arriba.

## Cursos (`courses`) — A

| Endpoint | Éxito | Errores |
| --- | --- | --- |
| `GET /courses` | `200` → `[]Course` (orden por nombre; vacío `[]`) | — |
| `POST /courses` | `201` → `Course` | `400` nombre vacío o de más de 40 caracteres · `405` |
| `GET /courses/{id}` | `200` → `Course` | `404` · `400` UUID inválido · `405` |
| `PUT /courses/{id}` | `200` → `Course` | `400` · `404` · `405` |
| `DELETE /courses/{id}` | `204` sin body | `404` · `400` UUID inválido · `405` |
| `GET /courses/{id}/semesters` | `200` → `[]Semester` (por año, descendente) | `400` UUID inválido · `405` |

Body de `POST` y `PUT`:

```json
{ "name": "Programación Web" }
```

`Course`:

```json
{ "id_course": "uuid", "name": "Programación Web" }
```

## Cursadas (`semesters`) — A

| Endpoint | Éxito | Errores |
| --- | --- | --- |
| `POST /semesters` | `201` → `Semester` | `400` faltantes, `end_date` mal formada o token de más de 32 caracteres · `409` token repetido o FK inexistente |
| `GET /semesters/{idOrToken}` | `200` → `Semester` (UUID) o `SemesterByToken` (token) | `404` |
| `GET /profesores/{id}/semesters` · `GET /semesters?professor_id=…` | `200` → `[]SemesterByProfessor` | `400` UUID faltante o inválido · `405` |
| `PUT /semesters/{id}` (cerrar o corregir fecha) | `200` → `Semester` | `400` `end_date` mal formada · `404` |
| `DELETE /semesters/{id}` | `204` sin body | `404` · `400` UUID inválido |

Body de `POST /semesters` (`sub_token` es opcional: si falta, se genera):

```json
{
  "id_course": "uuid",
  "professor_id": "uuid",
  "year": 2026,
  "sub_token": "opcional",
  "end_date": "2026-12-15"
}
```

Body de `PUT /semesters/{id}`:

```json
{ "end_date": "2026-12-15" }
```

`Semester` (forma cruda: `POST`, `PUT`, `GET` con UUID y listado por curso):

```json
{
  "id_semester": "uuid",
  "id_course": "uuid",
  "professor_id": "uuid",
  "year": 2026,
  "sub_token": "abc123",
  "end_date": "2026-12-15T00:00:00Z"
}
```

`SemesterByToken` (`GET /semesters/{sub_token}`): el anterior más `course_name`
e `is_open` (sección "Parte del PROFESOR (A)").

`SemesterByProfessor` (`GET /profesores/{id}/semesters` y
`GET /semesters?professor_id=…`):

```json
{
  "id_semester": "uuid",
  "year": 2026,
  "sub_token": "abc123",
  "end_date": "2026-12-15T00:00:00Z",
  "course_name": "Programación Web",
  "active_subscribers": 34
}
```

## Avisos (`notices`) — B (Fases 1-2)

| Endpoint | Éxito | Errores |
| --- | --- | --- |
| `GET /notices` | `200` → `[]Notice` (más reciente primero; vacío `[]`) | — |
| `POST /notices` | `201` → `Notice` | `400` título o mensaje vacíos, `id_semester` nulo, remitente inexistente · `405` |
| `GET /notices/{id}` | `200` → `Notice` | `404` · `400` UUID inválido · `405` |
| `PUT /notices/{id}` | `200` → `Notice` | `400` mensaje vacío · `404` · `405` |
| `DELETE /notices/{id}` | `204` sin body | `404` · `400` UUID inválido · `405` |

Body de `POST /notices`:

```json
{
  "id_semester": "uuid",
  "id_sender": "uuid",
  "title": "Parcial",
  "message": "Se toma el lunes"
}
```

Body de `PUT /notices/{id}` (solo `message`, decisión #7):

```json
{ "message": "Mensaje corregido" }
```

`Notice`:

```json
{
  "id_notice": "uuid",
  "id_semester": "uuid",
  "id_sender": "uuid",
  "title": "Parcial",
  "message": "Se toma el lunes",
  "created_at": "2026-10-18T10:00:00Z"
}
```

## Excepciones al "siempre JSON"

Fuera de los endpoints de la API, para que no sorprenda:

- `GET /` sirve `static/index.html` (HTML a propósito: es el frontend).
- `GET /health` responde `200` sin body, o `503` con texto plano
  (`database unavailable`): es infraestructura, no API.
- Las rutas no registradas caen en el `404 page not found` por defecto del
  `ServeMux` (texto plano).
- El index responde `405` con texto plano si el método no es `GET`.

Si se quiere todo JSON hasta en infraestructura, A puede cambiarlo sin tocar el
contrato de la API.
