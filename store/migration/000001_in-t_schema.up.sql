CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE "jam" (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    "owner_id" uuid NOT NULL,
    "bpm" uint NOT NULL DEFAULT 120,
    "capacity" uint NOT NULL DEFAULT 5,
    "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE temp TABLE IF NOT EXISTS "user" (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    username text UNIQUE NOT NULL CHECK (username <> ''),
    email citext UNIQUE NOT NULL CHECK (email ~ '^[a-zA-Z0-9.!#$%&’*+/=?^_\x60{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$'),
    PASSWORD citext NOT NULL CHECK (PASSWORD <> ''),
    created_at timestampz NOT NULL DEFAULT now()
);

