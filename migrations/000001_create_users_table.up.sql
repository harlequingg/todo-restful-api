CREATE TABLE IF NOT EXISTS users(
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    name varchar(255) NOT NULL,
    email citext UNIQUE NOT NULL,
    password_hash bytea NOT NULL,
    is_activated bool NOT NULL,
    version integer NOT NULL DEFAULT 1 
);