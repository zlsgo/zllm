package skill

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

func formatYAML(data interface{}) ([]byte, error) {
	return yaml.Marshal(data)
}

func formatJSON(data interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

func parseJSON(content []byte, target interface{}) error {
	return json.Unmarshal(content, target)
}
