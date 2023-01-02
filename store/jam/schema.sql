CREATE TABLE "jam" (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    "owner_id" uuid NOT NULL,
    "bpm" int NOT NULL DEFAULT 120,
    "capacity" int NOT NULL DEFAULT 5,
    "created_at" timestamptz NOT NULL DEFAULT (now())
);

