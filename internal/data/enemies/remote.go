package enemies

import (
	"context"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

type FileReader interface {
	ReadFile(
		ctx context.Context,
		filePath string,
	) ([]byte, error)
}

type RemoteLoader struct {
	files FileReader
}

func NewRemoteLoader(
	files FileReader,
) *RemoteLoader {
	return &RemoteLoader{
		files: files,
	}
}

func (l *RemoteLoader) LoadEnemies(
	ctx context.Context,
	language string,
	worldKey string,
) (
	coreEnemyFile,
	translatedEnemyFile,
	error,
) {
	corePath := fmt.Sprintf(
		"core/enemies/%s.yaml",
		worldKey,
	)

	translationPath := fmt.Sprintf(
		"i18n/%s/enemies/%s.json",
		language,
		worldKey,
	)

	coreRaw, err := l.files.ReadFile(
		ctx,
		corePath,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"load %s: %w",
			corePath,
			err,
		)
	}

	translationRaw, err := l.files.ReadFile(
		ctx,
		translationPath,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"load %s: %w",
			translationPath,
			err,
		)
	}

	var coreFile coreEnemyFile

	if err := yaml.Unmarshal(
		coreRaw,
		&coreFile,
	); err != nil {
		return nil, nil, fmt.Errorf(
			"parse %s: %w",
			corePath,
			err,
		)
	}

	var translations translatedEnemyFile

	if err := json.Unmarshal(
		translationRaw,
		&translations,
	); err != nil {
		return nil, nil, fmt.Errorf(
			"parse %s: %w",
			translationPath,
			err,
		)
	}

	return coreFile, translations, nil
}
