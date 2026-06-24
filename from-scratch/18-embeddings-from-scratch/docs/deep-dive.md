# 18-embeddings-from-scratch: Deep Dive

## What is an Embedding?

An embedding model is trained to map text to a point in high-dimensional space such that semantically similar text lands near each other.

```
"dog"   → point A in 1536-dimensional space
"puppy" → point B, very close to A
"car"   → point C, far from A and B
```

The model never "understands" language in a human sense — it learned statistical patterns from billions of text pairs during training.

## Why 1536 Dimensions?

`text-embedding-3-small` outputs 1536 floats. This is a design choice balancing:
- **Expressiveness:** more dimensions = can represent more nuanced relationships
- **Cost:** more dimensions = more storage + slower similarity search
- **Quality:** OpenAI found 1536 sufficient for most retrieval tasks

Each float32 is 4 bytes → 1536 × 4 = **6KB per embedding**.
1 million documents = 6GB just for embeddings.

## Cosine Similarity vs Euclidean Distance

Why cosine, not straight-line distance?

```
Document A: short paragraph  → embedding magnitude ≈ 0.1 (small vector)
Document B: long paragraph    → embedding magnitude ≈ 0.9 (large vector)

Same topic, different lengths → Euclidean distance is large (wrong)
                              → Cosine similarity is high (correct)
```

Cosine only cares about the **angle** between vectors, not their magnitude. Length of text doesn't affect similarity.

## The Semantic Search Pipeline

```
Query: "how do goroutines work?"
  ↓ embed with same model
Query vector: [0.12, -0.34, 0.89, ...]

For each document:
  similarity = cosineSimilarity(query_vec, doc_vec)

Results sorted by similarity:
  0.91 — "Goroutines are lightweight threads managed by Go runtime"
  0.84 — "Channels provide communication between goroutines"
  0.71 — "The Go scheduler uses M:N threading"
  0.21 — "Kubernetes pods are the smallest unit"  ← unrelated, filtered out
```

## Batch Embedding

This project embeds the whole corpus in one API call (`embedBatch`). The OpenAI Embeddings API accepts up to 2048 inputs per request. Always batch — individual calls have per-request overhead.

## What's Next (FS-19)

FS-18 stores vectors in a plain Go slice (`[][]float32`) and does a linear scan (O(n) per query). This works for 100 documents. For 100,000 documents you need an index. FS-19 builds that index from scratch before moving to pgvector in FS-20.
