CREATE extension IF NOT EXISTS "pgcrypto";
CREATE extension IF NOT EXISTS "citext";
CREATE temp TABLE IF NOT EXISTS "user" (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username text UNIQUE NOT NULL CHECK (username <> ''),
    email citext UNIQUE NOT NULL CHECK (
        email ~ '^[a-zA-Z0-9.!#$%&’*+/=?^_\x60{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$'
    ),
    PASSWORD citext NOT NULL CHECK (PASSWORD <> ''),
    created_at timestamp NOT NULL DEFAULT NOW()
);