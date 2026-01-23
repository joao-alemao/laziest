package flagparse

import (
	"strings"
)

// Flag represents a parsed flag from a command
type Flag struct {
	Name      string // "--config", "-v"
	Value     string // "100", "/path/to/file", "" for boolean
	IsBoolean bool   // true if no value or value is True/False
	StartIdx  int    // start index in original command (of flag name)
	EndIdx    int    // end index in original command (after value, or after flag if boolean)
}

// Parse extracts flags from a command string
// Returns the base command (everything before first flag) and list of flags
func Parse(command string) (string, []Flag) {
	tokens := tokenize(command)
	if len(tokens) == 0 {
		return command, nil
	}

	var flags []Flag
	var baseCmd string
	foundFirstFlag := false
	i := 0

	for i < len(tokens) {
		token := tokens[i]

		// Check if this token is a flag
		if isFlag(token.Value) {
			foundFirstFlag = true

			flag := Flag{
				Name:     token.Value,
				StartIdx: token.StartIdx,
				EndIdx:   token.EndIdx,
			}

			// Check if next token is a value (not another flag and exists)
			if i+1 < len(tokens) && !isFlag(tokens[i+1].Value) {
				valueToken := tokens[i+1]
				flag.Value = valueToken.Value
				flag.EndIdx = valueToken.EndIdx
				flag.IsBoolean = isBooleanValue(valueToken.Value)
				i += 2
			} else {
				// No value - boolean flag
				flag.IsBoolean = true
				i++
			}

			flags = append(flags, flag)
		} else {
			if !foundFirstFlag {
				// Part of base command
				if baseCmd == "" {
					baseCmd = token.Value
				} else {
					baseCmd += " " + token.Value
				}
			}
			// If we've found a flag but this isn't a flag, it was already consumed as a value
			i++
		}
	}

	// If no flags found, base command is the whole command
	if !foundFirstFlag {
		return command, nil
	}

	return strings.TrimSpace(baseCmd), flags
}

// token represents a token in the command with its position
type token struct {
	Value    string
	StartIdx int
	EndIdx   int
}

// tokenize splits a command into tokens, tracking positions
// Does not handle quoted strings with spaces (as per requirements)
func tokenize(command string) []token {
	var tokens []token
	start := -1

	for i, c := range command {
		if c == ' ' || c == '\t' {
			if start >= 0 {
				tokens = append(tokens, token{
					Value:    command[start:i],
					StartIdx: start,
					EndIdx:   i,
				})
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}

	// Last token
	if start >= 0 {
		tokens = append(tokens, token{
			Value:    command[start:],
			StartIdx: start,
			EndIdx:   len(command),
		})
	}

	return tokens
}

// isFlag checks if a token is a flag (starts with - or --)
func isFlag(s string) bool {
	return strings.HasPrefix(s, "-")
}

// isBooleanValue checks if a value is a boolean value
func isBooleanValue(s string) bool {
	lower := strings.ToLower(s)
	return lower == "true" || lower == "false"
}
