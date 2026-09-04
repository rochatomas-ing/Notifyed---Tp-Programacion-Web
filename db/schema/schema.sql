-- Script generado en RedGate DataModeler

-- tables
-- Table: courses
CREATE TABLE courses (
    id_course uuid  NOT NULL DEFAULT gen_random_uuid(),
    name varchar(40)  NOT NULL,
    CONSTRAINT course_pk PRIMARY KEY (id_course)
);

-- Table: inscriptions
CREATE TABLE inscriptions (
    id_inscription uuid  NOT NULL DEFAULT gen_random_uuid(),
    id_semester uuid  NOT NULL,
    email varchar(50)  NOT NULL,
    active boolean  NOT NULL,
    created_at timestamp  NOT NULL,
    CONSTRAINT inscription_pk PRIMARY KEY (id_inscription)
);

-- Table: notices
CREATE TABLE notices (
    id_notice uuid  NOT NULL DEFAULT gen_random_uuid(),
    id_semester uuid  NOT NULL,
    id_sender uuid  NOT NULL,
    title varchar(255)  NOT NULL,
    message text  NOT NULL,
    created_at timestamp  NOT NULL,
    CONSTRAINT notice_pk PRIMARY KEY (id_notice)
);

-- Table: semesters
CREATE TABLE semesters (
    id_semester uuid  NOT NULL DEFAULT gen_random_uuid(),
    id_course uuid  NOT NULL,
    professor_id uuid  NOT NULL,
    year int  NOT NULL,
    sub_token varchar(32)  NOT NULL,
    end_date date  NOT NULL,
    CONSTRAINT toke_sub_uk UNIQUE (sub_token) NOT DEFERRABLE  INITIALLY IMMEDIATE,
    CONSTRAINT semester_pk PRIMARY KEY (id_semester)
);

-- Table: users
CREATE TABLE users (
    id_user uuid  NOT NULL DEFAULT gen_random_uuid(),
    fullname varchar(50)  NOT NULL,
    email varchar(50)  NOT NULL,
    password_hash varchar(30)  NOT NULL,
    CONSTRAINT email_uk UNIQUE (email) NOT DEFERRABLE  INITIALLY IMMEDIATE,
    CONSTRAINT user_pk PRIMARY KEY (id_user)
);

-- foreign keys
-- Reference: inscription_semester (table: inscription)
ALTER TABLE inscriptions ADD CONSTRAINT inscription_semester
    FOREIGN KEY (id_semester)
    REFERENCES semester (id_semester)  
    NOT DEFERRABLE 
    INITIALLY IMMEDIATE
;

-- Reference: notice_semester (table: notice)
ALTER TABLE notices ADD CONSTRAINT notice_semester
    FOREIGN KEY (id_semester)
    REFERENCES semester (id_semester)  
    NOT DEFERRABLE 
    INITIALLY IMMEDIATE
;

-- Reference: notice_user (table: notice)
ALTER TABLE notices ADD CONSTRAINT notice_user
    FOREIGN KEY (id_sender)
    REFERENCES users (id_user)  
    NOT DEFERRABLE 
    INITIALLY IMMEDIATE
;

-- Reference: semester_course (table: semester)
ALTER TABLE semesters ADD CONSTRAINT semester_course
    FOREIGN KEY (id_course)
    REFERENCES course (id_course)  
    NOT DEFERRABLE 
    INITIALLY IMMEDIATE
;

-- Reference: semester_user (table: semester)
ALTER TABLE semesters ADD CONSTRAINT semester_user
    FOREIGN KEY (professor_id)
    REFERENCES users (id_user)  
    NOT DEFERRABLE 
    INITIALLY IMMEDIATE
;

-- End of file.

