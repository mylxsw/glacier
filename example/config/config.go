package config

import (
	"encoding/json"
)

type Config struct {
	DB   string
	Test string
}

func (conf Config) Serialize() string {
	data, _ := json.Marshal(conf)
	return string(data)
}
