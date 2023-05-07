CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE IF NOT EXISTS "user" (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    username varchar(255) NOT NULL CHECK (username <> ''),
    email citext UNIQUE NOT NULL CHECK (email ~ '^[a-zA-Z0-9.!#$%&’*+/=?^_\x60{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$'),
    created_at timestamptz NOT NULL DEFAULT (now())
);
