package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// UpdateInfo contains information about available updates
type UpdateInfo struct {
	Available   bool
	CurrentVer  string
	LatestVer   string
	ReleaseURL  string
	ReleaseName string
}

// parseVersion extracts version numbers from a version string (e.g., "v1.2.3" -> [1, 2, 3])
func parseVersion(version string) []int {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	nums := make([]int, len(parts))

	for i, part := range parts {
		// Remove any non-numeric suffixes (e.g., "1-beta", "2-rc1")
		numPart := strings.FieldsFunc(part, func(r rune) bool {
			return r == '-' || r == '+' || r == '_'
		})[0]

		if num, err := strconv.Atoi(numPart); err == nil {
			nums[i] = num
		}
	}

	return nums
}

// compareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	nums1 := parseVersion(v1)
	nums2 := parseVersion(v2)

	// Pad with zeros to make lengths equal
	maxLen := len(nums1)
	if len(nums2) > maxLen {
		maxLen = len(nums2)
	}

	for len(nums1) < maxLen {
		nums1 = append(nums1, 0)
	}
	for len(nums2) < maxLen {
		nums2 = append(nums2, 0)
	}

	// Compare each part
	for i := 0; i < maxLen; i++ {
		if nums1[i] < nums2[i] {
			return -1
		}
		if nums1[i] > nums2[i] {
			return 1
		}
	}

	return 0
}

// CheckForUpdates checks GitHub for the latest release of sshm
func CheckForUpdates(ctx context.Context, currentVersion string) (*UpdateInfo, error) {
	// Skip version check if current version is "dev"
	if currentVersion == "dev" {
		return &UpdateInfo{
			Available:  false,
			CurrentVer: currentVersion,
		}, nil
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.github.com/repos/Gu1llaum-3/sshm/releases/latest", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "sshm/"+currentVersion)

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Parse the response
	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Skip pre-releases and drafts
	if release.Prerelease || release.Draft {
		return &UpdateInfo{
			Available:  false,
			CurrentVer: currentVersion,
		}, nil
	}

	// Compare versions
	updateAvailable := compareVersions(currentVersion, release.TagName) < 0

	return &UpdateInfo{
		Available:   updateAvailable,
		CurrentVer:  currentVersion,
		LatestVer:   release.TagName,
		ReleaseURL:  release.HTMLURL,
		ReleaseName: release.Name,
	}, nil
}
