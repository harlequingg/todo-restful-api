CREATE TABLE IF NOT EXISTS tasks(
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    user_id int REFERENCES users(id) ON DELETE CASCADE,
    content text NOT NULL,
    is_completed boolean NOT NULL,
    version integer NOT NULL DEFAULT 1 
);