# 17 — Chat with Memory

Build a multi-turn chatbot. Understand why LLMs don't remember you by default, how conversation history works, and what happens when you hit the context limit.

**Prerequisites:** FS-16 (First LLM Call).

---

## The Problem: LLMs Are Stateless

Every API call is independent. The LLM has no memory of previous calls.

```mermaid
sequenceDiagram
    participant U as User
    participant APP as Your App
    participant LLM2 as LLM

    U->>APP: "My name is Alice"
    APP->>LLM2: [{"role":"user","content":"My name is Alice"}]
    LLM2-->>APP: "Nice to meet you, Alice!"

    U->>APP: "What's my name?"
    APP->>LLM2: [{"role":"user","content":"What's my name?"}]
    LLM2-->>APP: "I don't know your name."
    Note over LLM2: Previous call forgotten — new API call, no history
```

**The fix: send the whole conversation every time.**

```mermaid
sequenceDiagram
    participant U2 as User
    participant APP2 as Your App (with history)
    participant LLM3 as LLM

    U2->>APP2: "My name is Alice"
    APP2->>LLM3: [{user: "My name is Alice"}]
    LLM3-->>APP2: "Nice to meet you, Alice!"
    APP2->>APP2: history.append(user+assistant)

    U2->>APP2: "What's my name?"
    APP2->>LLM3: [{user:"My name is Alice"}, {assistant:"Nice to meet you, Alice!"}, {user:"What's my name?"}]
    LLM3-->>APP2: "Your name is Alice!"
```

---

## What You Build

A terminal chat application with:
- Persistent conversation history (in memory)
- System prompt (personality/role)
- Token counter — shows how much context is used
- Context window management — auto-truncate oldest messages when nearing the limit

```
You: My name is Alice and I'm a DevOps engineer
Assistant: Nice to meet you, Alice! How can I help with DevOps today?
[tokens: 45/128000]

You: What's my name and what do I do?
Assistant: You're Alice, a DevOps engineer.
[tokens: 98/128000]

You: /clear    ← reset history
You: /system "You are a snarky assistant"  ← change personality
You: /tokens   ← show current usage
```

---

## Context Window Management

```mermaid
graph LR
    FULL["History: 120K tokens<br>Approaching 128K limit"] --> STRAT{Strategy}
    STRAT -->|"Sliding window"| SLIDE["Drop oldest N messages<br>Keep system prompt + recent"]
    STRAT -->|"Summarize"| SUMM["Ask LLM to summarize<br>old messages first<br>Replace with summary"]
    STRAT -->|"Hard limit"| ERR["Return error to user<br>'Context full, start new chat'"]
```

## Key Concepts

| Concept | What it is |
|---------|-----------|
| **Message roles** | `system` (instructions), `user` (human), `assistant` (LLM response) |
| **Context window** | Total tokens (input + output) the model sees at once |
| **Sliding window** | Drop old messages to stay within limit |
| **System prompt** | First message that shapes all behavior, never dropped |

## Quick Start

```bash
export OPENAI_API_KEY=sk-...
make run
# Interactive terminal chat
```
