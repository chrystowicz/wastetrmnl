package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const OptionsPath = "/data/options.json"

type Entry struct {
	Date string `json:"date"`
	Hour string `json:"hour"`
}

type Options struct {
	TrmnlPluginUUID string  `json:"trmnl_plugin_uuid"`
	IdNumeru        string  `json:"id_numeru"`
	IdUlicy         string  `json:"id_ulicy"`
	Szop            []Entry `json:"szop"`
	Szot            []Entry `json:"szot"`
}

// Load reads Options from OptionsPath and validates required fields.
// Returns an error if the file is missing, malformed, or incomplete.
func Load() (*Options, error) {
	data, err := os.ReadFile(OptionsPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", OptionsPath, err)
	}

	var opts Options
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("parse %s: %w", OptionsPath, err)
	}

	if opts.TrmnlPluginUUID == "" {
		return nil, fmt.Errorf("%s: trmnl_plugin_uuid is required", OptionsPath)
	}

	return &opts, nil
}
