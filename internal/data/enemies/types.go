package enemies

import (
	"encoding/json"
	"fmt"
)

type School string

const (
	SchoolFire    School = "fire"
	SchoolIce     School = "ice"
	SchoolStorm   School = "storm"
	SchoolMyth    School = "myth"
	SchoolLife    School = "life"
	SchoolDeath   School = "death"
	SchoolBalance School = "balance"
	SchoolSun     School = "sun"
	SchoolMoon    School = "moon"
	SchoolStar    School = "star"
	SchoolShadow  School = "shadow"
	SchoolAny     School = "any"
)

type EnemyInfo struct {
	Name       string
	Affinities map[School]int
}

type coreEnemyFile map[string]coreEnemy

type coreEnemy struct {
	Affinities map[School]int `yaml:"affinities,omitempty"`
}

type translatedEnemyFile map[string]translatedEnemy

type translatedEnemy struct {
	Name string
}

func (e *translatedEnemy) UnmarshalJSON(
	data []byte,
) error {
	var name string

	if err := json.Unmarshal(
		data,
		&name,
	); err == nil {
		e.Name = name
		return nil
	}

	var object struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(
		data,
		&object,
	); err != nil {
		return fmt.Errorf(
			"enemy translation must be a string or object: %w",
			err,
		)
	}

	e.Name = object.Name

	return nil
}
