package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cosmos/dependency-guardian/pkg/analysis"
	"github.com/cosmos/dependency-guardian/pkg/config"
	"github.com/cosmos/dependency-guardian/pkg/github"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	ownerFlag     string
	repoFlag      string
	prNumberFlag  int
	noCommentFlag bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze dependencies in a pull request",
	Long: `Analyze the dependency impact of changes in a GitHub pull request.
This command will:
1. Fetch the changed files from the PR
2. Analyze the dependencies of changed packages
3. Show the impact on other packages in the repository`,
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	// CLI flags
	analyzeCmd.Flags().StringVarP(&ownerFlag, "owner", "o", "", "GitHub repository owner (overrides GITHUB_REPOSITORY if provided)")
	analyzeCmd.Flags().StringVarP(&repoFlag, "repo", "r", "", "GitHub repository name (overrides GITHUB_REPOSITORY if provided)")
	analyzeCmd.Flags().IntVarP(&prNumberFlag, "pr", "p", 0, "Pull request number (overrides PR_NUMBER if provided)")
	analyzeCmd.Flags().BoolVarP(&noCommentFlag, "no-comment", "n", false, "Do not post a comment on the PR")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	var cfg *config.Config
	var err error

	// If a config path is provided via flags, load it immediately.
	if cfgFile != "" {
		cfg, err = config.LoadConfig("", cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration from %s: %w", cfgFile, err)
		}
	}

	// Create GitHub client
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	client, err := github.NewClient()

	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	// Determine owner and repo
	var owner, repoName string

	if ownerFlag != "" && repoFlag != "" {
		owner = ownerFlag
		repoName = repoFlag
	} else {
		repoEnv := os.Getenv("GITHUB_REPOSITORY")
		if repoEnv == "" {
			return fmt.Errorf("either flags -o and -r must be provided or GITHUB_REPOSITORY env var must be set")
		}
		parts := strings.Split(repoEnv, "/")
		if len(parts) != 2 {
			return fmt.Errorf("GITHUB_REPOSITORY should be in the format 'owner/repo'")
		}
		owner, repoName = parts[0], parts[1]
		// Override with single flag if only one of them provided
		if ownerFlag != "" {
			owner = ownerFlag
		}
		if repoFlag != "" {
			repoName = repoFlag
		}
	}

	// Determine PR number
	var prNum int
	if prNumberFlag != 0 {
		prNum = prNumberFlag
	} else {
		prNumStr := os.Getenv("PR_NUMBER")
		if prNumStr == "" {
			return fmt.Errorf("either flag -p must be provided or PR_NUMBER env var must be set")
		}
		num, err := strconv.Atoi(prNumStr)
		if err != nil {
			return fmt.Errorf("invalid PR_NUMBER: %w", err)
		}
		prNum = num
	}

	// ------------------------------------------------------------------
	// Clone the repository at the PR head commit to a temporary directory
	// ------------------------------------------------------------------

	pr, err := client.GetPullRequest(owner, repoName, prNum)
	if err != nil {
		return fmt.Errorf("failed to fetch pull request: %w", err)
	}

	headRef := pr.GetHead().GetSHA()
	branchRef := pr.GetHead().GetRef() // e.g. feature/branch

	cloneDir, err := os.MkdirTemp("", "dep-guardian-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repoName)

	// Clone with depth 1 to target branch/ref
	cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", branchRef, repoURL, cloneDir)
	cloneOut, err := cloneCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %v\n%s", err, string(cloneOut))
	}

	// Ensure we are at the exact head SHA (in case branch moved)
	checkoutCmd := exec.Command("git", "-C", cloneDir, "checkout", headRef)
	checkoutOut, err := checkoutCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %v\n%s", err, string(checkoutOut))
	}

	workDir := cloneDir

	// If config wasn't loaded from a specific path, load it from the cloned repo.
	if cfg == nil {
		// The --config flag was not provided, so load from the default path in the repository.
		// cfgFile will be empty here.
		cfg, err = config.LoadConfig(workDir, cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	// Get root package path from the cloned repo's go.mod
	rootPkg, err := getRootPackage(workDir)
	if err != nil {
		return fmt.Errorf("failed to get root package from cloned repo: %w", err)
	}

	// Fetch changed files from PR
	files, err := client.GetPullRequestFiles(owner, repoName, prNum)
	if err != nil {
		return fmt.Errorf("failed to get PR files: %w", err)
	}

	// Convert to string slice (GitHub returns repo-relative paths)
	var changedFiles []string
	for _, file := range files {
		changedFiles = append(changedFiles, *file.Filename)
	}

	// Create analyzer
	analyzer := analysis.NewAnalyzer(cfg, workDir)
	analyzer.SetRootPackage(rootPkg)

	// Analyze changes
	result, err := analyzer.AnalyzeChangedPackages(changedFiles)
	if err != nil {
		return fmt.Errorf("failed to analyze changes: %w", err)
	}

	// Print results to stdout
	fmt.Println(result)

	// Post or update PR comment
	if !noCommentFlag {
		zap.S().Infow("posting or updating PR comment", "owner", owner, "repo", repoName, "pr", prNum)

		// Find existing comment
		var existingCommentID int64
		comments, err := client.ListComments(owner, repoName, prNum)
		if err != nil {
			return fmt.Errorf("failed to list PR comments: %w", err)
		}
		for _, comment := range comments {
			if strings.Contains(comment.GetBody(), "<!-- dependency-guardian -->") {
				existingCommentID = comment.GetID()
				break
			}
		}

		report := result.String()

		if existingCommentID != 0 {
			// Update existing comment
			zap.S().Infow("updating existing comment", "comment_id", existingCommentID)
			err = client.UpdateComment(owner, repoName, existingCommentID, report)
		} else {
			// Create new comment
			zap.S().Infow("creating new comment")
			err = client.CreateComment(owner, repoName, prNum, report)
		}

		if err != nil {
			return fmt.Errorf("failed to post or update PR comment: %w", err)
		}
	} else {
		zap.S().Infow("skipping PR comment due to --no-comment flag")
	}

	return nil
}

// getRootPackage gets the root package path from go.mod
func getRootPackage(dir string) (string, error) {
	modFile := filepath.Join(dir, "go.mod")
	content, err := os.ReadFile(modFile)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	// Extract module path from first line
	// Expected format: module github.com/org/repo
	var modulePath string
	_, err = fmt.Sscanf(string(content), "module %s", &modulePath)
	if err != nil {
		return "", fmt.Errorf("failed to parse go.mod: %w", err)
	}

	return modulePath, nil
} 