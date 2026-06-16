package repository

import (
	"context"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/notes"
)

type NoteRepo interface {
	List(ctx context.Context, userID, search, tags string, pinned *bool) ([]notes.Note, error)
	Get(ctx context.Context, id, userID string) (notes.Note, error)
	Create(ctx context.Context, n notes.Note) (notes.Note, error)
	Update(ctx context.Context, n notes.Note) (notes.Note, error)
	Delete(ctx context.Context, id, userID string) error
}
