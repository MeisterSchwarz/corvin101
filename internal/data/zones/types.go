package zones

import (
	"encoding/json"
	"fmt"
)

type ZoneInfo struct {
	Name  string `json:"name"`
	Sub   string `json:"sub,omitempty"`
	World string `json:"world"`
	Image string `json:"image"`

	Type       string `json:"type,omitempty"`
	Parent     string `json:"parent,omitempty"`
	Instance   bool   `json:"instance,omitempty"`
	Repeatable bool   `json:"repeatable,omitempty"`
}

type coreWorld struct {
	Image string `json:"image"`
	Order int    `json:"order"`
	Type  string `json:"type"`
}

type translatedWorld struct {
	Name string `json:"name"`
}

type coreZoneFile struct {
	Zones map[string]coreZone `json:"zones"`
}

type coreZone struct {
	Type       string `json:"type,omitempty"`
	Parent     string `json:"parent,omitempty"`
	Instance   bool   `json:"instance,omitempty"`
	Repeatable bool   `json:"repeatable,omitempty"`
}

type translatedZoneFile map[string]translatedZone

type translatedZone struct {
	Name string
	Sub  string
}

func (z *translatedZone) UnmarshalJSON(
	data []byte,
) error {
	var name string

	if err := json.Unmarshal(
		data,
		&name,
	); err == nil {
		z.Name = name
		z.Sub = ""

		return nil
	}

	var values []string

	if err := json.Unmarshal(
		data,
		&values,
	); err != nil {
		return fmt.Errorf(
			"translation must be a string or string array: %w",
			err,
		)
	}

	switch len(values) {
	case 1:
		z.Name = values[0]
		z.Sub = ""

		return nil

	case 2:
		z.Name = values[0]
		z.Sub = values[1]

		return nil

	default:
		return fmt.Errorf(
			"translation array must contain 1 or 2 strings, got %d",
			len(values),
		)
	}
}
