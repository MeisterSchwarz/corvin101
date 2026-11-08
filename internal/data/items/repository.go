package items

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sync"
)

type FileReader interface {
	ReadFile(context.Context, string) ([]byte, error)
}

// ID is a stable reference, independent of the translated display name.
type Info struct {
	ID   string
	Name string
}

type Repository struct {
	mu         sync.Mutex
	files      FileReader
	language   string
	categories map[string]map[string]Info
}

func NewRepository(files FileReader, language string) *Repository {
	return &Repository{files: files, language: language, categories: make(map[string]map[string]Info)}
}

func (r *Repository) Category(ctx context.Context, category string) (map[string]Info, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cached, ok := r.categories[category]; ok {
		return clone(cached), nil
	}
	raw, err := r.files.ReadFile(ctx, "core/items/"+category+".json")
	if err != nil {
		return nil, err
	}
	var core map[string]json.RawMessage
	if err = json.Unmarshal(raw, &core); err != nil {
		return nil, fmt.Errorf("parse items %s: %w", category, err)
	}
	raw, err = r.files.ReadFile(ctx, "i18n/"+r.language+"/items/"+category+".json")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	names := map[string]string{}
	if err == nil {
		if err = json.Unmarshal(raw, &names); err != nil {
			return nil, err
		}
	}
	result := make(map[string]Info, len(core))
	for id := range core {
		name := names[id]
		if name == "" {
			name = id
		}
		result[id] = Info{ID: id, Name: name}
	}
	r.categories[category] = result
	return clone(result), nil
}

func clone(source map[string]Info) map[string]Info {
	result := make(map[string]Info, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}
