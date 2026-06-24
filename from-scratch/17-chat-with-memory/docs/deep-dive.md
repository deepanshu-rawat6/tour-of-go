# 17-chat-with-memory: Deep Dive

## Why LLMs Don't Remember You

Every API call is completely independent. The LLM is a stateless function:

```
f(messages) → response
```

There is no session, no cookie, no server-side state. If you want it to remember, **you must send the entire conversation every time**.

## The messages Array

```json
[
  {"role": "system",    "content": "You are a helpful assistant."},
  {"role": "user",      "content": "My name is Alice"},
  {"role": "assistant", "content": "Nice to meet you, Alice!"},
  {"role": "user",      "content": "What is my name?"},
  {"role": "assistant", "content": "Your name is Alice."},
  {"role": "user",      "content": "What do I do for work?"}  ← new question
]
```

The entire array is sent on every request. The LLM reads all of it and "remembers" because it literally has all the text in its input.

## Context Window and Cost

Every message you send costs tokens — both old and new. If you have 10 turns of history:

```
Turn 1:  50 tokens sent
Turn 2:  120 tokens sent (turn 1 repeated + new turn)
Turn 3:  200 tokens sent
...
Turn 20: 2000+ tokens sent just for history
```

Cost grows with conversation length. For long sessions, you must either:
1. **Truncate:** drop oldest messages (this project — simplest)
2. **Summarize:** ask LLM to summarize old messages → replace with summary (better but uses a call)
3. **Hard limit:** tell user "context full, start new chat"

## The System Prompt

The system prompt is the first message with `role: "system"`. It:
- Sets the model's persona and behavior
- Is never shown to the end user (usually)
- Should be kept when truncating history

```
"You are a K8s expert who always answers with YAML examples."
"You are a customer support agent for Acme Corp. Never discuss competitors."
"You are a code reviewer. Be critical and point out every issue."
```

## Token Estimation vs Actual Count

This project estimates `len(text) / 4`. Real token count requires the tokenizer (tiktoken library). For most purposes, the estimate is good enough for truncation decisions — you don't need exact counts.

For exact counts: `github.com/pkoukk/tiktoken-go` implements the cl100k_base tokenizer used by GPT-4.
