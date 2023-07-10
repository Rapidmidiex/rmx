CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS "jam" (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name varchar(255) NOT NULL CHECK (name <> ''),
    bpm int NOT NULL DEFAULT 120 CHECK (bpm > 0),
    capacity int NOT NULL DEFAULT 5 CHECK (capacity > 0),
    created_at timestamptz NOT NULL DEFAULT (now())
);
