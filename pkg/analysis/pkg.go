package analysis

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Pkg represents a Go package and its dependencies
type Pkg struct {
	Name          string   // Package name (e.g., "github.com/org/repo/pkg/foo")
	Files         []string // Source files in this package
	Imports       []string // Direct imports
	Dependencies  []*Pkg   // Resolved dependency tree
	Internal     bool     // Whether this is an internal package
}

// Tree represents a package dependency tree
type Tree struct {
	Root        *Pkg              // Root package being analyzed
	Packages    map[string]*Pkg   // All packages in the tree
	RootDir     string           // Root directory of the project
	RootPkgPath string          // Root package path (e.g., "github.com/org/repo")
}

// NewTree creates a new dependency tree for analysis
func NewTree(rootDir, rootPkgPath string) *Tree {
	return &Tree{
		Packages:    make(map[string]*Pkg),
		RootDir:     rootDir,
		RootPkgPath: rootPkgPath,
	}
}

// Resolve builds the dependency tree for a given package
func (t *Tree) Resolve(pkgName string) error {
	// Check if we've already resolved this package
	if _, ok := t.Packages[pkgName]; ok {
		return nil // Already resolved
	}

	// Create new package
	pkg := &Pkg{
		Name:     pkgName,
		Internal: strings.HasPrefix(pkgName, t.RootPkgPath),
		Files:    make([]string, 0),
		Imports:  make([]string, 0),
	}
	t.Packages[pkgName] = pkg

	// Convert package path to filesystem path
	relPath := strings.TrimPrefix(pkgName, t.RootPkgPath)
	relPath = strings.TrimPrefix(relPath, "/")
	pkgPath := filepath.Join(t.RootDir, relPath)

	// Check if directory exists
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		zap.S().Warnw("package directory not found, skipping", "package", pkgName, "path", pkgPath)
		return nil
	}

	zap.S().Debugw("resolving dependencies for package", "package", pkgName, "path", pkgPath)

	// Parse package files
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgPath, nil, parser.ImportsOnly)
	if err != nil {
		return fmt.Errorf("failed to parse package %s at %s: %w", pkgName, pkgPath, err)
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("no Go packages found in directory %s", pkgPath)
	}

	// Track unique imports to avoid duplicates
	importSet := make(map[string]bool)

	// Collect all imports from all files in all packages
	for _, parsedPkg := range pkgs {
		for filename, file := range parsedPkg.Files {
			// Skip test files
			if strings.HasSuffix(filename, "_test.go") {
				continue
			}

			// Add the file to our list
			pkg.Files = append(pkg.Files, filename)

			// Process imports
			for _, imp := range file.Imports {
				// Remove quotes from import path
				importPath := strings.Trim(imp.Path.Value, "\"")

				// Only include internal imports and avoid duplicates
				if strings.HasPrefix(importPath, t.RootPkgPath) && !importSet[importPath] {
					importSet[importPath] = true
					pkg.Imports = append(pkg.Imports, importPath)

					// Recursively resolve the imported package
					if err := t.Resolve(importPath); err != nil {
						zap.S().Warnw("failed to resolve import, continuing", "import", importPath, "error", err)
						continue
					}

					// Add to dependencies
					if depPkg, ok := t.Packages[importPath]; ok {
						pkg.Dependencies = append(pkg.Dependencies, depPkg)
					}
				}
			}
		}
	}

	zap.S().Debugw("package processed", "package", pkgName, "files", len(pkg.Files), "imports", len(pkg.Imports))

	return nil
}

// FindReverseDependencies returns all packages that depend on the given package
func (t *Tree) FindReverseDependencies(pkgName string) []*Pkg {
	var deps []*Pkg
	for _, pkg := range t.Packages {
		// Skip the package itself
		if pkg.Name == pkgName {
			continue
		}

		// Check direct dependencies
		for _, dep := range pkg.Dependencies {
			if dep.Name == pkgName {
				deps = append(deps, pkg)
				break
			}
		}
	}

	zap.S().Debugw("found reverse dependencies", "for_package", pkgName, "count", len(deps))

	return deps
}

// IsInternal checks if a package is internal to the project
func (t *Tree) IsInternal(pkgName string) bool {
	return strings.HasPrefix(pkgName, t.RootPkgPath)
} 