CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    username TEXT UNIQUE NOT NULL CHECK (username <> ''),
    email TEXT UNIQUE NOT NULL CHECK (email ~ '^[a-zA-Z0-9.!#$%&’*+/=?^_\x60{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$'),
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    picture TEXT NOT NULL,
    blocked BOOLEAN NOT NULL DEFAULT FALSE,
    last_login TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

CREATE TABLE IF NOT EXISTS connections (
    provider_id TEXT UNIQUE NOT NULL,
    user_id uuid NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    CONSTRAINT FK_users_connections FOREIGN KEY(user_id) REFERENCES users(id)
)
