package skin

import (
	"encoding/json"
)

type ModelConfig struct {
	Default string `json:"default"`

	AnimatedFace string `json:"animated_face,omitempty"`
}

type modelConfigContainer struct {
	Geometry ModelConfig `json:"geometry"`
}

func (cfg ModelConfig) Encode() []byte {
	b, _ := json.Marshal(modelConfigContainer{Geometry: cfg})
	return b
}

func DecodeModelConfig(b []byte) (ModelConfig, error) {
	var m modelConfigContainer
	err := json.Unmarshal(b, &m)
	return m.Geometry, err
}
