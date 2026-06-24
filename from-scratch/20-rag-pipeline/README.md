# 16 — RAG Pipeline (Retrieval-Augmented Generation) From Scratch

Build a production RAG pipeline in Go — embedding, vector search, chunking, retrieval, and LLM-grounded answering. No LangChain, no Python. Just Go, pgvector, and the OpenAI API.

**SDE-2 AI/ML concepts demonstrated:**
- Text embedding and cosine similarity
- Vector database (pgvector) with IVFFlat index
- Document chunking with overlap
- RAG retrieval loop
- Prompt construction and LLM integration
- Streaming HTTP responses (SSE)
- Evaluation: retrieval score + LLM-as-judge

---

## Architecture

```mermaid
graph TD
    subgraph "Offline: Indexing Pipeline"
        DOCS["Source documents<br>(markdown, PDFs, text)"] --> CHUNK["Chunker<br>512 tokens, 50 overlap"]
        CHUNK --> EMBED["Embedder<br>POST /embeddings<br>text-embedding-3-small<br>float32[1536]"]
        EMBED --> STORE["pgvector<br>INSERT (content, embedding, metadata)"]
    end

    subgraph "Online: Query Pipeline"
        QUERY["User query:<br>'How does Raft work?'"] --> EMBED2["Embed query<br>same model"]
        EMBED2 --> SEARCH["Vector search<br>SELECT ... ORDER BY embedding <=> query_vec LIMIT 5"]
        SEARCH --> CHUNKS["Top 5 chunks<br>cosine similarity > 0.7"]
        CHUNKS --> PROMPT["Build prompt:<br>Context: [chunks]<br>Question: [query]"]
        PROMPT --> LLM["OpenAI gpt-4o-mini<br>streaming response"]
        LLM --> SSE["HTTP SSE stream<br>to client"]
    end
```

---

## How to Run

```bash
# Start PostgreSQL with pgvector
docker run -d \
  --name pgvector \
  -e POSTGRES_DB=rag \
  -e POSTGRES_PASSWORD=pass \
  -p 5432:5432 \
  pgvector/pgvector:pg16

export OPENAI_API_KEY=sk-...
export DATABASE_URL=postgres://postgres:pass@localhost:5432/rag?sslmode=disable

make build
make run

# Index some documents
curl -X POST http://localhost:8080/index \
  -H "Content-Type: application/json" \
  -d '{"path": "./docs/knowledge-base/"}'

# Query
curl -N http://localhost:8080/query?q=How+does+Raft+consensus+work
# -N disables buffering (needed for SSE)
```

---

## Key Concepts

- **Embedding** — a vector of floats representing the semantic meaning of text. Similar text has similar vectors (high cosine similarity).
- **Chunking with overlap** — split documents into 512-token windows with 50-token overlap. Overlap preserves context across chunk boundaries.
- **IVFFlat index** — approximate nearest-neighbour index. Divides vector space into `lists` (clusters). Query searches only nearest clusters. Fast for > 100K vectors.
- **Cosine similarity** — measures angle between vectors (not magnitude). `1.0` = identical, `0.0` = unrelated. pgvector `<=>` operator = cosine distance (1 - similarity).
- **Grounding** — LLM only sees retrieved context + question. Cannot hallucinate facts outside the provided chunks.
- **SSE (Server-Sent Events)** — HTTP streaming. Server pushes chunks of the LLM response as they arrive. Client renders progressively.

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md) — embedding math, cosine similarity derivation, IVFFlat index internals, chunking strategies, retrieval quality metrics
