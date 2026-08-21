package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// portPattern matches a YAML "port: N" line inside any config file.
var portPattern = regexp.MustCompile(`(?m)^\s*port:\s*(\d+)\s*$`)

// FindUsedPorts scans the workspace (outputDir/projectName/apps/*/config/config.yaml)
// and returns every port declared by existing applications.
func FindUsedPorts(outputDir, projectName string) (map[int]bool, error) {
	used := make(map[int]bool)
	if outputDir == "" || projectName == "" {
		return used, nil
	}

	appsDir := filepath.Join(outputDir, projectName, "apps")
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return used, nil // No workspace yet: nothing in use
		}
		return nil, fmt.Errorf("read workspace apps dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cfgFile := filepath.Join(appsDir, entry.Name(), "config", "config.yaml")
		data, err := os.ReadFile(cfgFile)
		if err != nil {
			continue // App has no config yet; nothing to avoid
		}
		for _, match := range portPattern.FindAllSubmatch(data, -1) {
			var p int
			if _, err := fmt.Sscanf(string(match[1]), "%d", &p); err == nil && p > 0 {
				used[p] = true
			}
		}
	}
	return used, nil
}

// AllocateFreePort returns the first free port starting at base, avoiding
// ports already used by existing applications in the workspace. This prevents
// runtime port conflicts when several generated apps run on the same host.
func AllocateFreePort(outputDir, projectName string, base int) (int, error) {
	if base <= 0 {
		base = 8080
	}
	used, err := FindUsedPorts(outputDir, projectName)
	if err != nil {
		return 0, fmt.Errorf("scan existing app ports: %w", err)
	}
	for p := base; p <= base+200; p++ {
		if !used[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free port found in range %d-%d", base, base+200)
}
