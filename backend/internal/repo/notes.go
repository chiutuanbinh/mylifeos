package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NoteRepo interface {
	List(ctx context.Context, userID, search, tags string, pinned *bool) ([]models.Note, error)
	Get(ctx context.Context, id, userID string) (models.Note, error)
	Create(ctx context.Context, n models.Note) (models.Note, error)
	Update(ctx context.Context, n models.Note) (models.Note, error)
	Delete(ctx context.Context, id, userID string) error
}

type pgNoteRepo struct{ db *pgxpool.Pool }

func NewNoteRepo(db *pgxpool.Pool) NoteRepo { return &pgNoteRepo{db} }

func (r *pgNoteRepo) List(ctx context.Context, userID, search, tags string, pinned *bool) ([]models.Note, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, content, tags, pinned, created_at, updated_at
		 FROM notes
		 WHERE user_id = $1
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%' OR content ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR $3 = ANY(tags))
		   AND ($4::boolean IS NULL OR pinned = $4)
		 ORDER BY pinned DESC, updated_at DESC`,
		userID, search, tags, pinned)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Note
	for rows.Next() {
		var n models.Note
		rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.Pinned, &n.CreatedAt, &n.UpdatedAt)
		out = append(out, n)
	}
	if out == nil {
		out = []models.Note{}
	}
	return out, rows.Err()
}

func (r *pgNoteRepo) Get(ctx context.Context, id, userID string) (models.Note, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, content, tags, pinned, created_at, updated_at
		 FROM notes WHERE id=$1 AND user_id=$2`, id, userID)
	var out models.Note
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.Content, &out.Tags, &out.Pinned, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *pgNoteRepo) Create(ctx context.Context, n models.Note) (models.Note, error) {
	if n.Tags == nil {
		n.Tags = []string{}
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO notes (user_id, title, content, tags, pinned)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, title, content, tags, pinned, created_at, updated_at`,
		n.UserID, n.Title, n.Content, n.Tags, n.Pinned)
	var out models.Note
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.Content, &out.Tags, &out.Pinned, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *pgNoteRepo) Update(ctx context.Context, n models.Note) (models.Note, error) {
	if n.Tags == nil {
		n.Tags = []string{}
	}
	row := r.db.QueryRow(ctx,
		`UPDATE notes SET title=$1, content=$2, tags=$3, pinned=$4, updated_at=now()
		 WHERE id=$5 AND user_id=$6
		 RETURNING id, user_id, title, content, tags, pinned, created_at, updated_at`,
		n.Title, n.Content, n.Tags, n.Pinned, n.ID, n.UserID)
	var out models.Note
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.Content, &out.Tags, &out.Pinned, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *pgNoteRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM notes WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}
