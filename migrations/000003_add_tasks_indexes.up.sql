CREATE INDEX IF NOT EXISTS tasks_content_index ON tasks USING GIN (to_tsvector('simple', content));