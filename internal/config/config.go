package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Command represents a saved command with its metadata
type Command struct {
	Name    string    `json:"name"`
	Command string    `json:"command"`
	Tags    []string  `json:"tags,omitempty"`
	AddedAt time.Time `json:"added_at"`
}

// Config holds all saved commands
type Config struct {
	Commands []Command `json:"commands"`
	path     string
}

// HistoryEntry represents a recently executed command
type HistoryEntry struct {
	Command   string    `json:"command"`   // Fully resolved command
	Name      string    `json:"name"`      // Original lz command name
	Timestamp time.Time `json:"timestamp"` // When it was executed
}

// GetConfigDir returns the path to the config directory
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "laziest"), nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "commands.json"), nil
}

// Load reads the config from disk
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Commands: []Command{},
		path:     configPath,
	}

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	cfg.path = configPath

	return cfg, nil
}

// Save writes the config to disk
func (c *Config) Save() error {
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// AddCommand adds a new command to the config
func (c *Config) AddCommand(name, command string, tags []string) error {
	// Check for duplicate names
	for _, cmd := range c.Commands {
		if cmd.Name == name {
			return fmt.Errorf("command with name '%s' already exists", name)
		}
	}

	// Validate tags
	for _, tag := range tags {
		if !IsValidTag(tag) {
			return fmt.Errorf("invalid tag '%s': must contain only letters, numbers, and underscores", tag)
		}
	}

	c.Commands = append(c.Commands, Command{
		Name:    name,
		Command: command,
		Tags:    tags,
		AddedAt: time.Now(),
	})

	return nil
}

// RemoveCommandByName removes a command by its name
func (c *Config) RemoveCommandByName(name string) error {
	for i, cmd := range c.Commands {
		if cmd.Name == name {
			c.Commands = append(c.Commands[:i], c.Commands[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("command '%s' not found", name)
}

// UpdateCommand updates an existing command by its original name
func (c *Config) UpdateCommand(originalName, newName, newCommand string, newTags []string) error {
	for i, cmd := range c.Commands {
		if cmd.Name == originalName {
			// Validate new tags
			for _, tag := range newTags {
				if !IsValidTag(tag) {
					return fmt.Errorf("invalid tag '%s': must contain only letters, numbers, and underscores", tag)
				}
			}
			c.Commands[i].Name = newName
			c.Commands[i].Command = newCommand
			c.Commands[i].Tags = newTags
			return nil
		}
	}
	return fmt.Errorf("command '%s' not found", originalName)
}

// GetCommandByName returns a command by its name
func (c *Config) GetCommandByName(name string) (*Command, error) {
	for i, cmd := range c.Commands {
		if cmd.Name == name {
			return &c.Commands[i], nil
		}
	}
	return nil, fmt.Errorf("command '%s' not found", name)
}

// GetCommandsByTag returns all commands that have the specified tag
func (c *Config) GetCommandsByTag(tag string) []Command {
	var result []Command
	for _, cmd := range c.Commands {
		for _, t := range cmd.Tags {
			if t == tag {
				result = append(result, cmd)
				break
			}
		}
	}
	return result
}

// GetAllTags returns a sorted list of all unique tags
func (c *Config) GetAllTags() []string {
	tagSet := make(map[string]struct{})
	for _, cmd := range c.Commands {
		for _, tag := range cmd.Tags {
			tagSet[tag] = struct{}{}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// GetTagCounts returns a map of tags to the number of commands with that tag
func (c *Config) GetTagCounts() map[string]int {
	counts := make(map[string]int)
	for _, cmd := range c.Commands {
		for _, tag := range cmd.Tags {
			counts[tag]++
		}
	}
	return counts
}

// IsValidTag checks if a tag name is valid (alphanumeric and underscores only, no spaces)
func IsValidTag(tag string) bool {
	if len(tag) == 0 {
		return false
	}

	for _, c := range tag {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// GetHistoryPath returns the path to the history file
func GetHistoryPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// LoadHistory reads the command history from disk
func LoadHistory() ([]HistoryEntry, error) {
	path, err := GetHistoryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// No history file yet, return empty slice
		return []HistoryEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read history: %w", err)
	}

	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse history: %w", err)
	}

	return entries, nil
}

// SaveHistory writes the command history to disk
func SaveHistory(entries []HistoryEntry) error {
	path, err := GetHistoryPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write history: %w", err)
	}

	return nil
}

// AddHistoryEntry adds a new command to the history (max 10, deduplicated)
func AddHistoryEntry(command, name string) error {
	// Load existing history
	entries, err := LoadHistory()
	if err != nil {
		return err
	}

	// Create new entry
	newEntry := HistoryEntry{
		Command:   command,
		Name:      name,
		Timestamp: time.Now(),
	}

	// Deduplicate: remove existing entry with same command
	filtered := []HistoryEntry{}
	for _, e := range entries {
		if e.Command != command {
			filtered = append(filtered, e)
		}
	}

	// Prepend new entry (most recent first)
	entries = append([]HistoryEntry{newEntry}, filtered...)

	// Trim to max 10
	if len(entries) > 10 {
		entries = entries[:10]
	}

	// Save
	return SaveHistory(entries)
}
