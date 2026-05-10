package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"tour_of_go/projects/triage-engine/internal/ports"
)

type KnowledgeRepo struct {
	pool *pgxpool.Pool
	llm  ports.LLMClient
}

func NewKnowledgeRepo(pool *pgxpool.Pool, llm ports.LLMClient) *KnowledgeRepo {
	return &KnowledgeRepo{pool: pool, llm: llm}
}

func (r *KnowledgeRepo) Search(ctx context.Context, query string, topK int) ([]string, error) {
	embedding, err := r.llm.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT content FROM runbook_chunks ORDER BY embedding <=> $1::vector LIMIT $2`,
		fmtVector(embedding), topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		chunks = append(chunks, content)
	}
	return chunks, rows.Err()
}

func (r *KnowledgeRepo) Index(ctx context.Context, docID, content string) error {
	embedding, err := r.llm.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("embed content: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO runbook_chunks (id, content, embedding) VALUES ($1, $2, $3::vector)
		 ON CONFLICT (id) DO UPDATE SET content = $2, embedding = $3::vector`,
		docID, content, fmtVector(embedding),
	)
	return err
}

// fmtVector converts []float32 to the pgvector literal format "[0.1,0.2,...]".
func fmtVector(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	s := "["
	for i, f := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%g", f)
	}
	return s + "]"
}
