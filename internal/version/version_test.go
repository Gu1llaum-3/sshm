package version

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected []int
	}{
		{"v1.2.3", []int{1, 2, 3}},
		{"1.2.3", []int{1, 2, 3}},
		{"v2.0.0", []int{2, 0, 0}},
		{"1.2.3-beta", []int{1, 2, 3}},
		{"1.2.3-rc1", []int{1, 2, 3}},
		{"dev", []int{0}},
	}

	for _, test := range tests {
		result := parseVersion(test.version)
		if len(result) != len(test.expected) {
			t.Errorf("parseVersion(%q) length = %d, want %d", test.version, len(result), len(test.expected))
			continue
		}
		for i, v := range result {
			if v != test.expected[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", test.version, i, v, test.expected[i])
				break
			}
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"v1.0.0", "v1.0.1", -1},
		{"v1.0.1", "v1.0.0", 1},
		{"v1.0.0", "v1.0.0", 0},
		{"1.2.3", "1.2.4", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.2.3-beta", "1.2.3", 0}, // Should ignore suffixes
		{"1.2.3", "1.2.3-rc1", 0},
	}

	for _, test := range tests {
		result := compareVersions(test.v1, test.v2)
		if result != test.expected {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", test.v1, test.v2, result, test.expected)
		}
	}
}
