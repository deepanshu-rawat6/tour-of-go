build:
	go build -o rag-pipeline .

run: build
	./rag-pipeline

test:
	go test -race -count=1 ./...

db-up:
	docker run -d --name pgvector -e POSTGRES_DB=rag -e POSTGRES_PASSWORD=pass \
		-p 5432:5432 pgvector/pgvector:pg16

db-schema:
	psql $$DATABASE_URL -c "CREATE EXTENSION IF NOT EXISTS vector; \
		CREATE TABLE IF NOT EXISTS chunks ( \
			id SERIAL PRIMARY KEY, source TEXT, chunk_index INT, \
			content TEXT, embedding vector(1536), created_at TIMESTAMPTZ DEFAULT now() \
		); \
		CREATE INDEX IF NOT EXISTS chunks_emb_idx ON chunks \
			USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);"
