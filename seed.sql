-- seed.sql — datos de prueba fijos para probar la API (ver requests.sh).
--
-- Carga:
--   bash seed.sh        (con la base levantada: make up)
--
-- Los UUID son fijos a propósito: requests.sh los usa hardcodeados.
-- ON CONFLICT DO NOTHING: se puede re-ejecutar sin duplicar filas, y restaura
-- lo que borran las pruebas (el aviso 5555… que borra el caso 5).
--
-- OJO: `make test` espera la base vacía. Correr los tests SIEMPRE antes de
-- cargar el seed (o hacer `make down && make up && make test`).

BEGIN;

-- El profesor (dueño de la cursada y de los avisos).
-- password_hash es cualquier valor: en este TP no hay login.
INSERT INTO users (id_user, fullname, email, password_hash)
VALUES ('11111111-1111-1111-1111-111111111111',
        'Profesor de Prueba',
        'profesor@test.com',
        'seed-no-hay-login-en-este-tp')
ON CONFLICT DO NOTHING;

-- La materia.
INSERT INTO courses (id_course, name)
VALUES ('22222222-2222-2222-2222-222222222222',
        'Programación Web')
ON CONFLICT DO NOTHING;

-- La cursada (vigente: end_date en el futuro). sub_token resuelve el QR del alumno.
INSERT INTO semesters (id_semester, id_course, professor_id, year, sub_token, end_date)
VALUES ('33333333-3333-3333-3333-333333333333',
        '22222222-2222-2222-2222-222222222222',
        '11111111-1111-1111-1111-111111111111',
        2026,
        'sub-token-de-prueba',
        '2026-12-15')
ON CONFLICT DO NOTHING;

-- Aviso fijo para los casos 3 y 4 (GET y PUT)...
INSERT INTO notices (id_notice, id_semester, id_sender, title, message)
VALUES ('44444444-4444-4444-4444-444444444444',
        '33333333-3333-3333-3333-333333333333',
        '11111111-1111-1111-1111-111111111111',
        'Aviso fijo (GET/PUT)',
        'Contenido original del seed.')
ON CONFLICT DO NOTHING;

-- ...y aviso fijo para los casos 5 y 6 (DELETE): el seed lo restaura al re-ejecutarse.
INSERT INTO notices (id_notice, id_semester, id_sender, title, message)
VALUES ('55555555-5555-5555-5555-555555555555',
        '33333333-3333-3333-3333-333333333333',
        '11111111-1111-1111-1111-111111111111',
        'Aviso fijo (DELETE)',
        'Este aviso lo borra la prueba del caso 5.')
ON CONFLICT DO NOTHING;

COMMIT;
