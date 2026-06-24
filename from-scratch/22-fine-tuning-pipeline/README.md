# 18 — Fine-Tuning Pipeline

Orchestrate the full fine-tuning workflow in Go — dataset preparation, job submission, monitoring, and model evaluation. Understand when RAG is not enough.

**Prerequisites:** FS-16 (RAG Pipeline) — fine-tuning and RAG are complementary.

---

## RAG vs Fine-Tuning — When to Use Which

```mermaid
graph TD
    Q{"What do you need?"}
    Q -->|"Access to private/current docs<br>Data changes frequently<br>Need to cite sources"| RAG["RAG (FS-16)<br>Better for knowledge injection"]
    Q -->|"Change HOW the model talks<br>Specific output format always<br>Domain-specific reasoning style<br>Reduce prompt length"| FT["Fine-Tuning (FS-18)<br>Better for behavior/style change"]
    Q -->|"Both: domain knowledge + style"| BOTH["RAG + Fine-tuned model"]
```

**Fine-tune when RAG isn't enough:**
- You want the model to always respond in a specific JSON schema without a long system prompt
- You're training on customer support transcripts to match your company's tone
- Domain vocabulary (medical, legal, code) that the base model doesn't understand well
- Reduce inference cost: smaller fine-tuned model replaces larger base model

---

## Architecture

```mermaid
graph LR
    RAW["Raw training data<br>(conversations, Q&A pairs)"] --> PREP["Dataset prep<br>JSONL format<br>messages: [{role, content}]"]
    PREP --> VALID["Validation<br>token count<br>format check<br>quality filter"]
    VALID --> UPLOAD["Upload to OpenAI<br>Files API"]
    UPLOAD --> JOB["Create fine-tuning job<br>base model + file ID"]
    JOB --> MONITOR["Poll job status<br>every 30s"]
    MONITOR -->|"succeeded"| EVAL["Evaluation<br>test prompts<br>compare base vs fine-tuned"]
    EVAL --> DEPLOY["Register model ID<br>use in production"]
```

---

## JSONL Training Format

```jsonl
{"messages": [{"role": "system", "content": "You are a K8s expert."}, {"role": "user", "content": "What is a Pod?"}, {"role": "assistant", "content": "A Pod is the smallest deployable unit in Kubernetes..."}]}
{"messages": [{"role": "user", "content": "How does HPA work?"}, {"role": "assistant", "content": "HPA scales replicas based on metrics..."}]}
```

## Quick Start

```bash
export OPENAI_API_KEY=sk-...
# Prepare your JSONL dataset in ./data/train.jsonl
make prepare   # validate + count tokens + estimate cost
make upload    # upload to OpenAI Files API
make train     # create fine-tuning job
make monitor   # poll until complete
make eval      # run test prompts against base + fine-tuned
```

## Key Concepts

- **LoRA (Low-Rank Adaptation)** — fine-tune only a small adapter matrix on top of frozen base weights. 100-1000x fewer trainable parameters than full fine-tune. Used internally by OpenAI for their fine-tuning API.
- **Token cost estimation** — fine-tuning costs per token. Count tokens before submitting.
- **Train/validation split** — hold out 10-20% for validation loss monitoring.
- **Overfitting** — if val loss rises while train loss drops, reduce epochs or data quality issues.
