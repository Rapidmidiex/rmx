CREATE TABLE users (
    id text NOT NULL PRIMARY KEY,
    username text NOT NULL,
    email text NOT NULL,
    password text NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    UNIQUE (email)
);
