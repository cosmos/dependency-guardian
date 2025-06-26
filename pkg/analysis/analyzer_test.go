package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/dependency-guardian/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeChangedPackages_SimpleDependency(t *testing.T) {
	// Setup a temporary repo structure where c depends on d
	repoPath := t.TempDir()
	rootPkg := "github.com/a/b"

	// Create go.mod
	goModPath := filepath.Join(repoPath, "go.mod")
	err := os.WriteFile(goModPath, []byte("module "+rootPkg), 0644)
	require.NoError(t, err)

	// Create package d
	pkgDPath := filepath.Join(repoPath, "d")
	err = os.MkdirAll(pkgDPath, 0755)
	require.NoError(t, err)

	dGoFile := filepath.Join(pkgDPath, "d.go")
	dGoContent := `package d

func D() {}`
	err = os.WriteFile(dGoFile, []byte(dGoContent), 0644)
	require.NoError(t, err)

	// Create package c that imports d
	pkgCPath := filepath.Join(repoPath, "c")
	err = os.MkdirAll(pkgCPath, 0755)
	require.NoError(t, err)

	cGoFile := filepath.Join(pkgCPath, "c.go")
	cGoContent := fmt.Sprintf(`package c

import "%s/d"

func C() {
	d.D()
}`, rootPkg)
	err = os.WriteFile(cGoFile, []byte(cGoContent), 0644)
	require.NoError(t, err)

	// Initialize analyzer
	cfg := config.DefaultConfig()
	cfg.Critical.Packages = []string{
		"**/c", // Mark package c as critical
	}
	analyzer := NewAnalyzer(cfg, repoPath)
	analyzer.SetRootPackage(rootPkg)

	// Simulate a change in package d
	changedFiles := []string{"d/d.go"}

	// Analyze
	result, err := analyzer.AnalyzeChangedPackages(changedFiles)
	require.NoError(t, err)

	// Print report
	t.Logf("Analysis Report:\n%s", result.String())

	// Assertions
	require.Len(t, result.Impacts, 1, "Should be one changed package impact")

	impact := result.Impacts[0]
	require.Equal(t, rootPkg+"/d", impact.ChangedPackage, "Changed package should be d")

	require.Len(t, impact.AffectedPackages, 1, "Should be one affected package")
	affectedPkg := impact.AffectedPackages[0]
	require.Equal(t, rootPkg+"/c", affectedPkg.Name, "Affected package should be c")
	require.True(t, affectedPkg.IsCritical, "Affected package c should be marked as critical")
} 