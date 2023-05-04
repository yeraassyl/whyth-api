package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type PresetType int

const (
	Toggle PresetType = iota
	Format
)

type Preset struct {
	Name    string     `json:"name"`
	Value   string     `json:"value"`
	Type    PresetType `json:"type"`
	Checked bool       `json:"checked"`
}

type PresetStore struct {
	presets []Preset
}

func (ps *PresetStore) loadPresets(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	var presets []Preset
	err = json.Unmarshal(data, &presets)
	if err != nil {
		return err
	}
	ps.presets = presets
	return nil
}

func (ps *PresetStore) GetPresets() ([]Preset, error) {
	if ps.presets == nil {
		return nil, fmt.Errorf("presets not loaded")
	}
	return ps.presets, nil
}
