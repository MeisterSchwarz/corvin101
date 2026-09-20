package zones

import (
	"context"
	"encoding/json"
	"fmt"
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

func (l *RemoteLoader) LoadWorldMetadata(
	ctx context.Context,
	language string,
) (
	map[string]coreWorld,
	map[string]translatedWorld,
	error,
) {
	corePath :=
		"core/worlds.json"

	translationPath :=
		fmt.Sprintf(
			"i18n/%s/worlds.json",
			language,
		)

	coreRaw, err :=
		l.files.ReadFile(
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

	translationRaw, err :=
		l.files.ReadFile(
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

	var coreWorlds map[string]coreWorld

	if err := json.Unmarshal(
		coreRaw,
		&coreWorlds,
	); err != nil {
		return nil, nil, fmt.Errorf(
			"parse %s: %w",
			corePath,
			err,
		)
	}

	var translatedWorlds map[string]translatedWorld

	if err := json.Unmarshal(
		translationRaw,
		&translatedWorlds,
	); err != nil {
		return nil, nil, fmt.Errorf(
			"parse %s: %w",
			translationPath,
			err,
		)
	}

	return coreWorlds, translatedWorlds, nil
}

func (l *RemoteLoader) LoadWorldZones(
	ctx context.Context,
	language string,
	worldKey string,
) (
	coreZoneFile,
	translatedZoneFile,
	error,
) {
	corePath :=
		fmt.Sprintf(
			"core/zones/%s.json",
			worldKey,
		)

	translationPath :=
		fmt.Sprintf(
			"i18n/%s/zones/%s.json",
			language,
			worldKey,
		)

	coreRaw, err :=
		l.files.ReadFile(
			ctx,
			corePath,
		)

	if err != nil {
		return coreZoneFile{},
			nil,
			fmt.Errorf(
				"load %s: %w",
				corePath,
				err,
			)
	}

	translationRaw, err :=
		l.files.ReadFile(
			ctx,
			translationPath,
		)

	if err != nil {
		return coreZoneFile{},
			nil,
			fmt.Errorf(
				"load %s: %w",
				translationPath,
				err,
			)
	}

	var coreFile coreZoneFile

	if err := json.Unmarshal(
		coreRaw,
		&coreFile,
	); err != nil {
		return coreZoneFile{},
			nil,
			fmt.Errorf(
				"parse %s: %w",
				corePath,
				err,
			)
	}

	var translations translatedZoneFile

	if err := json.Unmarshal(
		translationRaw,
		&translations,
	); err != nil {
		return coreZoneFile{},
			nil,
			fmt.Errorf(
				"parse %s: %w",
				translationPath,
				err,
			)
	}

	return coreFile, translations, nil
}
