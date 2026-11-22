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
	Gold  *GoldRange
}

type GoldRange struct{ Min, Max int }

type reference struct {
	Item string `yaml:"item"`
}

// Accept concise IDs while retaining compatibility with existing item objects.
func (r *reference) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		r.Item = node.Value
	} else if node.Kind == yaml.MappingNode {
		type plain reference
		if err := node.Decode((*plain)(r)); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("item reference must be a string or an item object")
	}
	if r.Item == "" {
		return fmt.Errorf("empty item reference")
	}
	return nil
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
	var source map[string]map[string]yaml.Node
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
			node := categories[key]
			if key == "gold" {
				if node.Kind != yaml.SequenceNode || len(node.Content) != 2 || node.Content[0].Tag != "!!int" || node.Content[1].Tag != "!!int" {
					return nil, fmt.Errorf("gold for %s must contain exactly two integer amounts", enemy)
				}
				var amounts []int
				if err := node.Decode(&amounts); err != nil {
					return nil, fmt.Errorf("gold for %s: %w", enemy, err)
				}
				if len(amounts) != 2 || amounts[0] < 0 || amounts[1] < amounts[0] {
					return nil, fmt.Errorf("gold for %s must be [minimum, maximum] with 0 <= minimum <= maximum", enemy)
				}
				result[enemy] = append(result[enemy], Category{Key: "gold", Name: "Gold", Gold: &GoldRange{Min: amounts[0], Max: amounts[1]}})
				continue
			}
			var references []reference
			if err := node.Decode(&references); err != nil {
				return nil, fmt.Errorf("%s drops for %s: %w", key, enemy, err)
			}
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
			for _, ref := range references {
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
		if result[i].Gold != nil {
			gold := *result[i].Gold
			result[i].Gold = &gold
		}
		result[i].Items = slices.Clone(result[i].Items)
	}
	return result
}
