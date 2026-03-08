package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Background string  `toml:"background" json:"background"`
	Groups     []Group `toml:"group"      json:"groups"`
}

type Group struct {
	Name  string `toml:"name"  json:"name"`
	Links []Link `toml:"link"  json:"links"`
	X     int    `toml:"x"     json:"x"`
	Y     int    `toml:"y"     json:"y"`
	W     int    `toml:"w"     json:"w"`
}

type Link struct {
	Name string `toml:"name" json:"name"`
	URL  string `toml:"url"  json:"url"`
	Icon string `toml:"icon" json:"icon"`
}

const defaultConfig = `background = "#0f0f1a"

[[group]]
name = "Dev"

  [[group.link]]
  name = "GitHub"
  url = "https://github.com"
  icon = ""

  [[group.link]]
  name = "Go Docs"
  url = "https://pkg.go.dev"
  icon = "📖"
`

func EnsureExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.WriteFile(path, []byte(defaultConfig), 0644)
	}
	return nil
}

func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
