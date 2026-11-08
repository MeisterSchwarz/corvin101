package drops

import (
	"context"
	"corvin101/internal/data/items"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"slices"
	"sort"
)

type Category struct {
	Key   string
	Name  string
	Items []items.Info
}

type reference struct {
	Item string `yaml:"item"`
}

type Repository struct {
	files items.FileReader
	items *items.Repository
}

func NewRepository(files items.FileReader, language string) *Repository {
	return &Repository{files: files, items: items.NewRepository(files, language)}
}

func (r *Repository) LoadWorld(ctx context.Context, world string) (map[string][]Category, error) {
	path := "core/drops/" + world + ".yaml"
	raw, err := r.files.ReadFile(ctx, path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var source map[string]map[string][]reference
	if err = yaml.Unmarshal(raw, &source); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	result := make(map[string][]Category, len(source))
	for enemy, categories := range source {
		keys := make([]string, 0, len(categories))
		for key := range categories {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			catalog, err := r.items.Category(ctx, key)
			if err != nil {
				return nil, fmt.Errorf("load drop category %s: %w", key, err)
			}
			name := key
			if key == "treasurecards" {
				name = "Goldkarten"
			}
			category := Category{Key: key, Name: name}
			seen := make(map[string]bool)
			for _, ref := range categories[key] {
				item, ok := catalog[ref.Item]
				if !ok {
					return nil, fmt.Errorf("unknown item reference %q for %s", ref.Item, enemy)
				}
				if !seen[item.ID] {
					category.Items = append(category.Items, item)
					seen[item.ID] = true
				}
			}
			sort.Slice(category.Items, func(i, j int) bool { return category.Items[i].Name < category.Items[j].Name })
			if len(category.Items) > 0 {
				result[enemy] = append(result[enemy], category)
			}
		}
	}
	return result, nil
}

func Clone(source []Category) []Category {
	result := slices.Clone(source)
	for i := range result {
		result[i].Items = slices.Clone(result[i].Items)
	}
	return result
}
