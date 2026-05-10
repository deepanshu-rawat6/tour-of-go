# Stateful Support/Alert Triage Engine

A Go-based LangGraph-equivalent workflow engine for triaging support tickets with human-in-the-loop approval.

**SDE 2 concepts demonstrated:** State persistence · Human-in-the-loop (HITL) · RAG with pgvector · LLM integration · Hexagonal architecture · Async workflows

## Architecture

```
POST /webhooks/ticket
        │
        ▼
  TriageEngine.Start()
        │
        ├─ categorize      → LLM (OpenAI) classifies ticket category
        ├─ retrieveRunbook → pgvector cosine search for relevant runbook chunks
        ├─ executeDiagnostic → HTTP call to CI/diagnostic API
        ├─ draftResolution → LLM drafts response using runbook + diagnostic
        └─ awaitHuman      → SAVE STATE to PostgreSQL + notify engineer
                                        [PAUSED — state survives restarts]

POST /graph/resume {ticket_id, approved: true/false}
        │
        ▼
  TriageEngine.Resume()
        │
        ├─ Load state from PostgreSQL
        ├─ approved=true  → StatusCompleted
        └─ approved=false → StatusRejected
```

## Quick Start

```bash
make docker-up   # start pgvector/pgvector:pg16 on :5432
make run         # start server on :8080
```

## Demo: Full Triage Flow

```bash
# 1. Submit a ticket
curl -s -X POST localhost:8080/webhooks/ticket \
  -H "Content-Type: application/json" \
  -d '{
    "ID": "JIRA-123",
    "Summary": "Build pipeline stuck on step 3",
    "Description": "The CI pipeline has been stuck for 2 hours",
    "Reporter": "alice",
    "Priority": "high"
  }'
# Response: {"ticket_id":"JIRA-123","status":"awaiting_human"}
# Server logs the approval request with the drafted response.

# 2. Engineer approves
curl -s -X POST localhost:8080/graph/resume \
  -H "Content-Type: application/json" \
  -d '{"ticket_id":"JIRA-123","approved":true}'
# Response: {"ticket_id":"JIRA-123","status":"completed","drafted_response":"..."}

# 3. Or reject
curl -s -X POST localhost:8080/graph/resume \
  -H "Content-Type: application/json" \
  -d '{"ticket_id":"JIRA-123","approved":false}'
# Response: {"ticket_id":"JIRA-123","status":"rejected","drafted_response":"..."}
```

## Testing

```bash
make test                # unit tests — no DB, no OpenAI (all mocked)
make docker-up
make integration-test    # integration tests with testcontainers (spins up pgvector automatically)
```

## Configuration

Set `CONFIG_PATH` to a YAML file:

```yaml
server:
  addr: ":8080"
database:
  dsn: "postgres://triage:triage@localhost:5432/triage"
openai:
  apiKey: "sk-..."
  model: "gpt-4o-mini"
  embeddingModel: "text-embedding-3-small"
diagnostic:
  baseURL: "http://your-jenkins:8080"
```

## Key Design Decisions

- **State as JSONB** — `InvestigationState` is serialized to a single JSONB column. Simple, queryable, no migration needed when fields are added.
- **Pause/resume via DB** — The graph pauses by saving state and returning. Resume loads from DB. The process can restart between Start and Resume — state survives.
- **pgvector for RAG** — Runbook chunks are embedded with `text-embedding-3-small` and stored as `vector(1536)`. Cosine similarity search (`<=>`) finds the top-K relevant chunks.
- **LLMClient port** — The graph nodes never import the openai package. Swapping to a local model (Ollama) requires only a new adapter.
- **No graph framework** — The "graph" is a sequential function call chain with a single pause point. This is all LangGraph does for a linear workflow — no framework needed.
- **testcontainers-go** — Integration tests spin up `pgvector/pgvector:pg16` automatically. No manual Docker setup required for CI.
