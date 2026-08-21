package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed all:templates/make
var makeTemplates embed.FS

// MakeSyncConfig holds the configuration for Makefile/scripts synchronization.
type MakeSyncConfig struct {
	WorkspacePath string
	AppName       string // empty = all apps under apps/
	DryRun        bool
}

// MakeSyncChange describes one planned change.
type MakeSyncChange struct {
	App    string
	Kind   string // gen-script | thin-target | add-target | phony | help
	Detail string
}

// MakeSyncResult reports the sync outcome.
type MakeSyncResult struct {
	Apps    []string
	Changes []MakeSyncChange
}

// standardScripts maps standard script names to embedded templates.
// Every standard script must be invoked via a thin Makefile target.
var standardScripts = []string{"run", "test", "test-coverage", "clean"}

var (
	makeTargetRe = regexp.MustCompile(`^([a-zA-Z0-9_-]+):`)
	phonyRe      = regexp.MustCompile(`^\.PHONY:`)
)

// Sync analyzes each app's scripts/ and Makefile, then (unless dry-run)
// generates missing standard scripts and rewrites inline targets as thin
// script wrappers, appends missing targets, and updates .PHONY.
func (c *MakeSyncConfig) Sync() (*MakeSyncResult, error) {
	workspace, err := resolveWorkspacePath(c.WorkspacePath)
	if err != nil {
		return nil, err
	}

	apps, err := listApps(workspace, c.AppName)
	if err != nil {
		return nil, err
	}

	result := &MakeSyncResult{Apps: apps}
	for _, appName := range apps {
		changes, err := syncOneApp(workspace, appName, c.DryRun)
		if err != nil {
			return nil, fmt.Errorf("app %s: %w", appName, err)
		}
		result.Changes = append(result.Changes, changes...)
	}
	return result, nil
}

func listApps(workspace, appName string) ([]string, error) {
	if appName != "" {
		if _, err := os.Stat(filepath.Join(workspace, "apps", appName)); err != nil {
			return nil, fmt.Errorf("app %q not found under apps/", appName)
		}
		return []string{appName}, nil
	}
	entries, err := os.ReadDir(filepath.Join(workspace, "apps"))
	if err != nil {
		return nil, fmt.Errorf("cannot list apps: %w", err)
	}
	var apps []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(workspace, "apps", e.Name(), "Makefile")); err == nil {
			apps = append(apps, e.Name())
		}
	}
	sort.Strings(apps)
	if len(apps) == 0 {
		return nil, fmt.Errorf("no apps with a Makefile found under apps/")
	}
	return apps, nil
}

func syncOneApp(workspace, appName string, dryRun bool) ([]MakeSyncChange, error) {
	appDir := filepath.Join(workspace, "apps", appName)
	scriptsDir := filepath.Join(appDir, "scripts")
	makefilePath := filepath.Join(appDir, "Makefile")

	content, err := os.ReadFile(makefilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read Makefile: %w", err)
	}
	lines := strings.Split(string(content), "\n")

	targets := map[string]int{} // name -> line index
	for i, ln := range lines {
		if m := makeTargetRe.FindStringSubmatch(ln); m != nil {
			targets[m[1]] = i
		}
	}

	existingScripts := map[string]bool{}
	if entries, err := os.ReadDir(scriptsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
				existingScripts[strings.TrimSuffix(e.Name(), ".sh")] = true
			}
		}
	}

	var changes []MakeSyncChange

	// 1. Generate missing standard scripts.
	for _, name := range standardScripts {
		if existingScripts[name] {
			continue
		}
		changes = append(changes, MakeSyncChange{App: appName, Kind: "gen-script", Detail: name + ".sh"})
		if !dryRun {
			if err := writeStandardScript(scriptsDir, name); err != nil {
				return nil, err
			}
		}
	}

	// 2. Thin out inline standard targets (keep the ## comment).
	// Process in descending line order: forward edits would shift later
	// targets' line numbers.
	type thinJob struct {
		name string
		idx  int
	}
	var jobs []thinJob
	for _, name := range standardScripts {
		idx, ok := targets[name]
		if !ok {
			continue
		}
		if strings.HasPrefix(lines[idx], name+": ##") && idx+1 < len(lines) && strings.HasPrefix(lines[idx+1], "\t@$(SCRIPTS_DIR)/") {
			continue // already a thin wrapper
		}
		jobs = append(jobs, thinJob{name: name, idx: idx})
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].idx > jobs[j].idx })
	for _, job := range jobs {
		name, idx := job.name, job.idx
		changes = append(changes, MakeSyncChange{App: appName, Kind: "thin-target", Detail: name})
		if dryRun {
			continue
		}
		comment := lines[idx]
		end := len(lines)
		for j := idx + 1; j < len(lines); j++ {
			if makeTargetRe.MatchString(lines[j]) {
				end = j
				break
			}
		}
		block := []string{
			comment,
			"\t@$(SCRIPTS_DIR)/" + name + ".sh",
			"",
		}
		lines = append(lines[:idx], append(block, lines[end:]...)...)
	}

	// 3. Append targets for scripts without a Makefile target.
	// Multi-command scripts (e.g. migrate.sh driven by migrate-up/migrate-down)
	// are already referenced — only append a target when no target references
	// the script at all.
	var missing []string
	for script := range existingScripts {
		if _, ok := targets[script]; ok {
			continue
		}
		referenced := false
		for _, ln := range lines {
			if strings.Contains(ln, "$(SCRIPTS_DIR)/"+script+".sh") {
				referenced = true
				break
			}
		}
		if !referenced {
			missing = append(missing, script)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		changes = append(changes, MakeSyncChange{App: appName, Kind: "add-target", Detail: strings.Join(missing, ", ")})
		if !dryRun {
			lines = append(lines, "")
			for _, name := range missing {
				lines = append(lines,
					name+": ## 脚本包装（ygctl make sync 生成）",
					"\t@$(SCRIPTS_DIR)/"+name+".sh",
					"",
				)
			}
		}
	}

	// 4. Update .PHONY for all script-backed targets.
	seenNames := map[string]bool{}
	var phonyNames []string
	for _, s := range append(append([]string{}, standardScripts...), missing...) {
		if !seenNames[s] {
			seenNames[s] = true
			phonyNames = append(phonyNames, s)
		}
	}
	if len(phonyNames) > 0 {
		addedCount := 0
		if !dryRun {
			lines, addedCount = rewritePhony(lines, phonyNames)
		} else {
			addedCount = countPhonyMissing(lines, phonyNames)
		}
		if addedCount > 0 {
			changes = append(changes, MakeSyncChange{App: appName, Kind: "phony", Detail: strings.Join(phonyNames, ", ")})
		}
	}

	if !dryRun {
		if err := os.WriteFile(makefilePath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return nil, fmt.Errorf("cannot write Makefile: %w", err)
		}
	}
	return changes, nil
}

// writeStandardScript renders one standard script template.
func writeStandardScript(scriptsDir, name string) error {
	content, err := makeTemplates.ReadFile(filepath.Join("templates", "make", name+".sh.tmpl"))
	if err != nil {
		return fmt.Errorf("no standard template for script %s: %w", name, err)
	}
	outPath := filepath.Join(scriptsDir, name+".sh")
	if err := os.WriteFile(outPath, content, 0755); err != nil {
		return err
	}
	return nil
}

// rewritePhony merges script-backed names into the existing .PHONY lines
// (first .PHONY line absorbs new names, others are left untouched).
// Returns the updated lines and the number of names actually added.
func rewritePhony(lines []string, names []string) ([]string, int) {
	added := map[string]bool{}
	addedCount := 0
	for i, ln := range lines {
		if !phonyRe.MatchString(ln) {
			continue
		}
		fields := strings.Fields(ln)[1:]
		existing := map[string]bool{}
		for _, f := range fields {
			existing[f] = true
		}
		var toAdd []string
		for _, n := range names {
			if !existing[n] && !added[n] {
				toAdd = append(toAdd, n)
				added[n] = true
			}
		}
		if len(toAdd) > 0 {
			lines[i] = strings.TrimRight(ln, " \t") + " " + strings.Join(toAdd, " ")
			addedCount += len(toAdd)
		}
	}
	return lines, addedCount
}

// countPhonyMissing counts how many names would be added (dry-run).
func countPhonyMissing(lines []string, names []string) int {
	_, n := rewritePhony(append([]string{}, lines...), names)
	return n
}
