# 19-vector-store-from-scratch: Deep Dive

## What This Implements

A flat (brute-force) vector store — the same algorithm inside pgvector's exact search, Pinecone's smallest tier, and every vector DB when the index hasn't warmed up yet.

## Brute-Force Search — O(n) per Query

```
For each query:
  1. Embed the query text → query_vector
  2. For every stored record:
       score = cosineSimilarity(query_vector, record.vector)
  3. Sort by score descending
  4. Return top K

Time: O(n × d) where n=records, d=vector dimensions (1536)
1000 records × 1536 dims = 1.5M multiplications per query — fast on modern CPU
100K records → 150M multiplications → ~50ms — starts to feel slow
1M records → needs an index (see FS-20 / pgvector IVFFlat)
```

## Metadata Filtering

Metadata filtering runs BEFORE similarity search — reduces the set of vectors to compare:

```
/search?q=kubernetes+errors&source=k8s-docs&version=1.29

1. Filter: keep only records where metadata["source"]="k8s-docs" AND metadata["version"]="1.29"
2. Search: cosine similarity only against filtered subset
```

This is exactly how pgvector's `WHERE` clause + `ORDER BY embedding <=>` works.

## Binary Persistence Format

Each record serialized as:
```
[id_length: 4 bytes uint32][id: variable]
[text_length: 4 bytes][text: variable]
[vec_length: 4 bytes][float32 × 1536: 6144 bytes]
[meta_json_length: 4 bytes][meta JSON: variable]
```

Total per record: ~6KB (dominated by the vector).
10K records ≈ 60MB on disk.

## Why Build This Before Using pgvector?

After building this, pgvector's IVFFlat index makes intuitive sense:
- IVFFlat clusters all vectors into N groups at index creation time (k-means)
- At query time: only search the nearest M clusters instead of all records
- This project = search all records. IVFFlat = search M/N fraction of records.

The algorithm is identical — the index just lets you skip most comparisons.

## What pgvector Adds That This Doesn't

| Feature | This project | pgvector |
|---------|-------------|---------|
| Storage | In-memory (lost on crash) | PostgreSQL ACID |
| Index | None (full scan) | IVFFlat, HNSW |
| Scale | ~100K records | Millions |
| Filtering | In-memory map lookup | SQL WHERE |
| Transactions | No | Yes |
| Replication | No | PostgreSQL streaming |
