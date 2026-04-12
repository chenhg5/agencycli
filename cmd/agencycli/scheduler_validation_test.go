package main

import (
	"strings"
	"testing"
)

func TestValidateWakeupCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		wantErr   bool
		errContains string
	}{
		// Valid conditions
		{
			name:      "empty condition",
			condition: "",
			wantErr:   false,
		},
		{
			name:      "gh issue list",
			condition: "gh issue list --state open --json id --jq 'length > 0'",
			wantErr:   false,
		},
		{
			name:      "gh pr list with grep",
			condition: "gh pr list --state open | grep -q .",
			wantErr:   false,
		},
		{
			name:      "agencycli inbox with grep",
			condition: "agencycli --dir $AGENCY_DIR inbox messages --unread-only | grep -q .",
			wantErr:   false,
		},
		{
			name:      "git status check",
			condition: "git status --porcelain | grep -q .",
			wantErr:   false,
		},
		{
			name:      "true command",
			condition: "true",
			wantErr:   false,
		},
		{
			name:      "false command",
			condition: "false",
			wantErr:   false,
		},
		{
			name:      "test command",
			condition: "test -f /tmp/flag",
			wantErr:   false,
		},
		{
			name:      "jq with safe env",
			condition: "jq -r '.count' $AGENCY_DIR/stats.json",
			wantErr:   false,
		},

		// Invalid conditions - dangerous metacharacters
		{
			name:      "semicolon injection",
			condition: "gh issue list; rm -rf /",
			wantErr:   true,
			errContains: "blocked pattern",
		},
		{
			name:      "AND operator injection",
			condition: "true && rm -rf /",
			wantErr:   true,
			errContains: "blocked pattern",
		},
		{
			name:      "OR operator injection",
			condition: "false || curl malicious.com",
			wantErr:   true,
			errContains: "blocked pattern",
		},
		{
			name:      "command substitution",
			condition: "echo $(whoami)",
			wantErr:   true,
			errContains: "blocked pattern",
		},
		{
			name:      "backtick substitution",
			condition: "echo `whoami`",
			wantErr:   true,
			errContains: "blocked pattern",
		},
		{
			name:      "output redirection",
			condition: "gh issue list > /tmp/issues",
			wantErr:   true,
			errContains: "blocked pattern",
		},
		{
			name:      "input redirection",
			condition: "grep pattern < /etc/passwd",
			wantErr:   true,
			errContains: "blocked pattern",
		},
		{
			name:      "background execution",
			condition: "gh issue list &",
			wantErr:   true,
			errContains: "blocked pattern",
		},
		{
			name:      "newline injection",
			condition: "gh issue list\nrm -rf /",
			wantErr:   true,
			errContains: "blocked pattern",
		},

		// Invalid conditions - disallowed commands
		{
			name:      "curl command",
			condition: "curl https://example.com",
			wantErr:   true,
			errContains: "allowed command",
		},
		{
			name:      "rm command",
			condition: "rm -rf /tmp/data",
			wantErr:   true,
			errContains: "allowed command",
		},
		{
			name:      "echo command",
			condition: "echo hello",
			wantErr:   true,
			errContains: "allowed command",
		},
		{
			name:      "cat command",
			condition: "cat /etc/passwd",
			wantErr:   true,
			errContains: "allowed command",
		},
		{
			name:      "python command",
			condition: "python3 script.py",
			wantErr:   true,
			errContains: "allowed command",
		},
		{
			name:      "bash command",
			condition: "bash script.sh",
			wantErr:   true,
			errContains: "allowed command",
		},
		{
			name:      "sh command",
			condition: "sh script.sh",
			wantErr:   true,
			errContains: "allowed command",
		},

		// Invalid conditions - unsafe env vars
		{
			name:      "unsafe env var HOME",
			condition: "gh issue list --dir $HOME",
			wantErr:   true,
			errContains: "unsafe env var",
		},
		{
			name:      "unsafe env var PATH",
			condition: "grep -r $PATH",
			wantErr:   true,
			errContains: "unsafe env var",
		},
		{
			name:      "unsafe env var USER",
			condition: "jq '$USER'",
			wantErr:   true,
			errContains: "unsafe env var",
		},

		// Edge cases
		{
			name:      "empty string after pipe",
			condition: "gh issue list |",
			wantErr:   true,
			errContains: "empty pipe segment",
		},
		{
			name:      "disallowed after pipe",
			condition: "gh issue list | curl https://evil.com",
			wantErr:   true,
			errContains: "allowed command",
		},
		{
			name:      "multiple pipes all allowed",
			condition: "gh issue list --json id | jq 'length' | grep -q '1'",
			wantErr:   false,
		},
		{
			name:      "multiple pipes one disallowed",
			condition: "gh issue list | grep pattern | rm -rf /tmp",
			wantErr:   true,
			errContains: "allowed command",
		},
		{
			name:      "disallowed at third position",
			condition: "gh issue list | jq '.id' | bash script.sh",
			wantErr:   true,
			errContains: "allowed command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWakeupCondition(tt.condition)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateWakeupCondition(%q) expected error, got nil", tt.condition)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateWakeupCondition(%q) error should contain '%s', got: %v", tt.condition, tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("validateWakeupCondition(%q) unexpected error: %v", tt.condition, err)
				}
			}
		})
	}
}