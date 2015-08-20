package main

import "testing"

func TestIsValidCommitSha(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"099c693933ef19b7258b91cfbb245bbe1748d307", true},
		{"dd6a843a1d08d673ebd580d5507e4fa2b90dd507", true},
		{"dd6a84", false},
		{"dd6a843a1d08d6", false},
		{"echo foobar && lol", false},
		{"rm -rf", false},
		{"master", false},
		{"develop", false},
	}

	for _, tt := range tests {
		got := isValidCommitSha(tt.input)
		if got != tt.expected {
			t.Errorf("isValidCommit test fail. input=%s, want=%v, got=%v", tt.input, tt.expected, got)
		}
	}
}
