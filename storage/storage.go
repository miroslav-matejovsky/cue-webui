package storage

import "context"

// Store loads persisted form values and saves updated values after submission.
// Values are keyed by the dot-separated field path used by the HTML form.
type Store interface {
	Load(ctx context.Context) (map[string]string, error)
	Save(ctx context.Context, values map[string]string) error
}
