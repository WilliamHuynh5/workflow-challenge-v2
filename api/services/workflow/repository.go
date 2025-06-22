package workflow

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	query := `SELECT id, name, definition, created_at, updated_at FROM workflows WHERE id = $1`
	var wf Workflow
	var def []byte
	if err := r.pool.QueryRow(ctx, query, id).Scan(&wf.ID, &wf.Name, &def, &wf.CreatedAt, &wf.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(def, &wf.Definition); err != nil {
		return nil, err
	}
	return &wf, nil
}

func (r *Repository) SaveWorkflow(ctx context.Context, wf *Workflow) error {
	def, err := json.Marshal(wf.Definition)
	if err != nil {
		return err
	}
	query := `INSERT INTO workflows (id, name, definition) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, definition = EXCLUDED.definition, updated_at = NOW()`
	_, err = r.pool.Exec(ctx, query, wf.ID, wf.Name, def)
	return err
}
