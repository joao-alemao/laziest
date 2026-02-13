package flagparse

import (
	"testing"
)

func TestParseSegments(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []Segment
	}{
		{
			name:    "command with subcommand between flags",
			command: `watch -n 10 aws ec2 start-instances --instance-ids "i-0c7b" --profile ai-dev/Admin`,
			expected: []Segment{
				{Type: SegmentStatic, Static: "watch"},
				{Type: SegmentFlag, Flag: &Flag{Name: "-n", Value: "10", IsBoolean: false}},
				{Type: SegmentStatic, Static: "aws ec2 start-instances"},
				{Type: SegmentFlag, Flag: &Flag{Name: "--instance-ids", Value: `"i-0c7b"`, IsBoolean: false}},
				{Type: SegmentFlag, Flag: &Flag{Name: "--profile", Value: "ai-dev/Admin", IsBoolean: false}},
			},
		},
		{
			name:    "simple command with flags at end",
			command: "git commit -m hello --amend",
			expected: []Segment{
				{Type: SegmentStatic, Static: "git commit"},
				{Type: SegmentFlag, Flag: &Flag{Name: "-m", Value: "hello", IsBoolean: false}},
				{Type: SegmentFlag, Flag: &Flag{Name: "--amend", Value: "", IsBoolean: true}},
			},
		},
		{
			name:    "no flags",
			command: "echo hello world",
			expected: []Segment{
				{Type: SegmentStatic, Static: "echo hello world"},
			},
		},
		{
			name:    "flags only",
			command: "--config test.yaml --verbose",
			expected: []Segment{
				{Type: SegmentFlag, Flag: &Flag{Name: "--config", Value: "test.yaml", IsBoolean: false}},
				{Type: SegmentFlag, Flag: &Flag{Name: "--verbose", Value: "", IsBoolean: true}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := ParseSegments(tt.command)

			if len(segments) != len(tt.expected) {
				t.Errorf("expected %d segments, got %d", len(tt.expected), len(segments))
				for i, seg := range segments {
					if seg.Type == SegmentStatic {
						t.Logf("  %d: [Static] %q", i, seg.Static)
					} else {
						t.Logf("  %d: [Flag] %s = %q", i, seg.Flag.Name, seg.Flag.Value)
					}
				}
				return
			}

			for i, seg := range segments {
				exp := tt.expected[i]
				if seg.Type != exp.Type {
					t.Errorf("segment %d: expected type %v, got %v", i, exp.Type, seg.Type)
					continue
				}

				if seg.Type == SegmentStatic {
					if seg.Static != exp.Static {
						t.Errorf("segment %d: expected static %q, got %q", i, exp.Static, seg.Static)
					}
				} else {
					if seg.Flag.Name != exp.Flag.Name {
						t.Errorf("segment %d: expected flag name %q, got %q", i, exp.Flag.Name, seg.Flag.Name)
					}
					if seg.Flag.Value != exp.Flag.Value {
						t.Errorf("segment %d: expected flag value %q, got %q", i, exp.Flag.Value, seg.Flag.Value)
					}
					if seg.Flag.IsBoolean != exp.Flag.IsBoolean {
						t.Errorf("segment %d: expected IsBoolean %v, got %v", i, exp.Flag.IsBoolean, seg.Flag.IsBoolean)
					}
				}
			}
		})
	}
}
