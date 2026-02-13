package flagparse

import (
	"strings"
)

// SegmentType indicates whether a segment is static text or a flag
type SegmentType int

const (
	SegmentStatic SegmentType = iota // Non-flag portion: "watch", "aws ec2 start-instances"
	SegmentFlag                      // Flag with optional value: "-n 10", "--profile ai-dev/Admin"
)

// Flag represents a parsed flag from a command
type Flag struct {
	Name      string // "--config", "-v"
	Value     string // "100", "/path/to/file", "" for boolean
	IsBoolean bool   // true if no value or value is True/False
}

// Segment represents a portion of a command, either static text or a flag
type Segment struct {
	Type   SegmentType
	Static string // For SegmentStatic: the static text
	Flag   *Flag  // For SegmentFlag: the flag details
}

// ParseSegments parses a command into an ordered list of segments
// This preserves the relative order of all command parts, allowing commands like:
// "watch -n 10 aws ec2 start-instances --instance-ids i-123"
// to be correctly parsed with "aws ec2 start-instances" in its original position
func ParseSegments(command string) []Segment {
	tokens := tokenize(command)
	if len(tokens) == 0 {
		return nil
	}

	var segments []Segment
	var staticAccumulator []string
	i := 0

	for i < len(tokens) {
		tok := tokens[i]

		if isFlag(tok.Value) {
			// Flush accumulated static tokens as a Static segment
			if len(staticAccumulator) > 0 {
				segments = append(segments, Segment{
					Type:   SegmentStatic,
					Static: strings.Join(staticAccumulator, " "),
				})
				staticAccumulator = nil
			}

			// Parse the flag
			flag := &Flag{
				Name: tok.Value,
			}

			// Check if next token is a value (not another flag and exists)
			if i+1 < len(tokens) && !isFlag(tokens[i+1].Value) {
				valueToken := tokens[i+1]
				flag.Value = valueToken.Value
				flag.IsBoolean = isBooleanValue(valueToken.Value)
				i += 2
			} else {
				// No value - boolean flag
				flag.IsBoolean = true
				i++
			}

			segments = append(segments, Segment{
				Type: SegmentFlag,
				Flag: flag,
			})
		} else {
			// Not a flag - accumulate as static text
			staticAccumulator = append(staticAccumulator, tok.Value)
			i++
		}
	}

	// Flush any remaining static tokens
	if len(staticAccumulator) > 0 {
		segments = append(segments, Segment{
			Type:   SegmentStatic,
			Static: strings.Join(staticAccumulator, " "),
		})
	}

	return segments
}

// HasFlags returns true if the segments contain at least one flag
func HasFlags(segments []Segment) bool {
	for _, seg := range segments {
		if seg.Type == SegmentFlag {
			return true
		}
	}
	return false
}

// token represents a token in the command with its position
type token struct {
	Value    string
	StartIdx int
	EndIdx   int
}

// tokenize splits a command into tokens, tracking positions
// Does not handle quoted strings with spaces
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
