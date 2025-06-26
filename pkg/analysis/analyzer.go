package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cosmos/dependency-guardian/pkg/config"
)

// AffectedPackage represents a package that is impacted by a change.
type AffectedPackage struct {
	Name       string
	IsCritical bool
}

// PackageImpact details the packages affected by a change in a single package.
type PackageImpact struct {
	ChangedPackage   string
	AffectedPackages []*AffectedPackage
}

// AnalysisResult contains the results of dependency analysis
type AnalysisResult struct {
	Impacts              []*PackageImpact
	DirectDependencies   []string
	IndirectDependencies []string
}

// Analyzer handles dependency analysis for a repository
type Analyzer struct {
	cfg        *config.Config
	tree       *Tree
	repoPath   string
	rootPkgPath string
}

// NewAnalyzer creates a new analyzer instance
func NewAnalyzer(cfg *config.Config, repoPath string) *Analyzer {
	return &Analyzer{
		cfg:      cfg,
		repoPath: repoPath,
	}
}

// SetRootPackage sets the root package path for the analyzer
func (a *Analyzer) SetRootPackage(rootPkg string) {
	a.rootPkgPath = rootPkg
	a.tree = NewTree(a.repoPath, rootPkg)
}

// AnalyzeChangedPackages analyzes the dependencies of changed packages
func (a *Analyzer) AnalyzeChangedPackages(changedFiles []string) (*AnalysisResult, error) {
	if a.tree == nil {
		return nil, fmt.Errorf("analyzer not initialized with root package")
	}

	// First, resolve all packages in the repository to build a complete dependency graph
	err := filepath.Walk(a.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Check for .go files to identify a package directory
			goFiles, _ := filepath.Glob(filepath.Join(path, "*.go"))
			if len(goFiles) > 0 {
				relPath, err := filepath.Rel(a.repoPath, path)
				if err != nil {
					return err
				}
				pkgPath := filepath.ToSlash(relPath)
				if pkgPath == "." {
					// skip root, it's not a real package in this context
					return nil
				}
				fullPkgPath := a.rootPkgPath + "/" + pkgPath
				if err := a.tree.Resolve(fullPkgPath); err != nil {
					// Log a warning but continue analysis
					fmt.Printf("Warning: failed to resolve dependencies for %s: %v\n", fullPkgPath, err)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking repository: %w", err)
	}

	// Track unique packages
	changedPkgs := make(map[string]bool)

	// First pass: identify changed packages
	for _, file := range changedFiles {
		if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			continue
		}

		pkgPath := filepath.Dir(file)
		var fullPkgPath string
		if pkgPath == "." {
			fullPkgPath = a.rootPkgPath
		} else {
			fullPkgPath = a.rootPkgPath + "/" + pkgPath
		}
		changedPkgs[fullPkgPath] = true
	}

	// Second pass: find impacts for each changed package
	var impacts []*PackageImpact
	allAffectedPkgs := make(map[string]bool)

	var sortedChangedPkgs []string
	for pkg := range changedPkgs {
		sortedChangedPkgs = append(sortedChangedPkgs, pkg)
	}
	sort.Strings(sortedChangedPkgs)

	for _, pkgName := range sortedChangedPkgs {
		revDeps := a.tree.FindReverseDependencies(pkgName)
		var affectedForPkg []*AffectedPackage
		for _, dep := range revDeps {
			if a.cfg.ShouldIgnorePackage(dep.Name) {
				continue
			}

			// Only include affected packages that are also high-level targets
			if !a.cfg.IsHighLevelPackage(dep.Name) {
				continue
			}

			affectedPkg := &AffectedPackage{
				Name:       dep.Name,
				IsCritical: a.cfg.IsCriticalPackage(dep.Name),
			}

			affectedForPkg = append(affectedForPkg, affectedPkg)
			allAffectedPkgs[dep.Name] = true
		}

		sort.Slice(affectedForPkg, func(i, j int) bool {
			return affectedForPkg[i].Name < affectedForPkg[j].Name
		})

		impacts = append(impacts, &PackageImpact{
			ChangedPackage:   pkgName,
			AffectedPackages: affectedForPkg,
		})
	}

	// Re-calculate direct and indirect dependencies for the summary
	directDeps := make(map[string]bool)
	for _, pkgName := range sortedChangedPkgs {
		if p, ok := a.tree.Packages[pkgName]; ok {
			for _, dep := range p.Dependencies {
				directDeps[dep.Name] = true
			}
		}
	}

	var directDepList []string
	for dep := range directDeps {
		directDepList = append(directDepList, dep)
	}

	var indirectDepList []string
	for pkg := range allAffectedPkgs {
		if !directDeps[pkg] {
			indirectDepList = append(indirectDepList, pkg)
		}
	}

	sort.Strings(directDepList)
	sort.Strings(indirectDepList)

	// Build result
	result := &AnalysisResult{
		Impacts:              impacts,
		DirectDependencies:   directDepList,
		IndirectDependencies: indirectDepList,
	}

	return result, nil
}

// String returns a string representation of the analysis result
func (r *AnalysisResult) String() string {
	var b strings.Builder
	b.WriteString("<!-- dependency-guardian -->\n")
	b.WriteString("## 🔍 Dependency Impact Analysis\n\n")

	if len(r.Impacts) == 0 {
		b.WriteString("No changed packages found.\n")
		return b.String()
	}

	b.WriteString("### Changed Packages and Their Impacts\n\n")
	for _, impact := range r.Impacts {
		b.WriteString(fmt.Sprintf("#### Changed Package: `%s`\n\n", impact.ChangedPackage))
		if len(impact.AffectedPackages) > 0 {
			summary := fmt.Sprintf("<details><summary>Affected Packages (%d)</summary>\n\n", len(impact.AffectedPackages))
			b.WriteString(summary)
			for _, pkg := range impact.AffectedPackages {
				if pkg.IsCritical {
					b.WriteString(fmt.Sprintf("- 🚨 **`%s`** (Critical)\n", pkg.Name))
				} else {
					b.WriteString(fmt.Sprintf("- `%s`\n", pkg.Name))
				}
			}
			b.WriteString("\n</details>\n\n")
		} else {
			b.WriteString("This change does not affect any other packages.\n\n")
		}
	}

	b.WriteString("### Analysis Summary:\n\n")

	totalChanged := len(r.Impacts)
	totalAffected := 0
	affectedSet := make(map[string]bool)
	for _, impact := range r.Impacts {
		for _, pkg := range impact.AffectedPackages {
			if !affectedSet[pkg.Name] {
				affectedSet[pkg.Name] = true
				totalAffected++
			}
		}
	}

	b.WriteString(fmt.Sprintf("- **Changed packages**: %d\n", totalChanged))
	b.WriteString(fmt.Sprintf("- **Affected packages**: %d\n", totalAffected))
	b.WriteString(fmt.Sprintf("- **Direct dependencies of changed packages**: %d\n", len(r.DirectDependencies)))
	b.WriteString(fmt.Sprintf("- **Indirectly affected packages**: %d\n", len(r.IndirectDependencies)))

	return b.String()
}