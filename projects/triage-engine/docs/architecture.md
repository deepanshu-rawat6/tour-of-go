# Architecture

## Overview

Stateful support ticket triage engine implementing a LangGraph-equivalent workflow with human-in-the-loop (HITL) approval.

## Components

```
cmd/main.go                      → HTTP server (webhook + resume endpoints)
internal/engine/engine.go        → Graph execution engine (node traversal)
internal/nodes/                  → Individual graph nodes (categorize, retrieve, diagnose, draft, await)
internal/store/state.go          → PostgreSQL state persistence (JSONB)
internal/rag/retriever.go        → pgvector-based runbook retrieval
migrations/                      → Schema (investigation_states, runbook_chunks)
```

## Workflow

1. Webhook receives support ticket
2. Engine executes nodes sequentially: categorize → retrieveRunbook → executeDiagnostic → draftResolution → awaitHuman
3. At `awaitHuman`, state is serialized to PostgreSQL (JSONB) and process pauses
4. Engineer reviews draft, calls `POST /graph/resume` with approval/rejection
5. Engine loads state from DB and completes the workflow

## Key Concepts

- **State machine persistence**: Process can crash/restart between Start and Resume
- **pgvector RAG**: Runbook chunks embedded as vectors, retrieved via cosine similarity
- **HITL**: Human approval gate before automated resolution
- **testcontainers-go**: Integration tests spin up pgvector:pg16 automatically
