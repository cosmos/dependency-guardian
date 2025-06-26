# Dependency Guardian

A CLI tool and GitHub Action that analyzes Pull Requests to identify dependency tree changes and their potential impact on other packages in the repository. It helps developers understand the full scope of their changes and ensures proper testing of affected components by posting a comment on the PR.

## Features

- Analyzes modified files in GitHub PRs to identify dependency impacts
- Generates reverse dependency graphs showing affected high-level modules
- Integrates seamlessly into any pull request workflow as a GitHub Action
- Provides configurable filtering and customization options
- Posts a clear, actionable impact analysis as a PR comment

## Usage

This tool is designed to be used as a GitHub Action.

1.  Add a workflow file to your repository (e.g., `.github/workflows/dependency-guardian.yml`):

    ```yaml
    name: 'Dependency Guardian'
    on: [pull_request]

    jobs:
      analyze:
        name: Analyze Dependencies
        runs-on: ubuntu-latest
        permissions:
          contents: read
          pull-requests: write # Required to post a comment
        steps:
          - uses: actions/checkout@v4
          - name: Run Dependency Guardian
            uses: cosmos/dependency-guardian@v1 # Use the latest major version
            with:
              github-token: ${{ secrets.GITHUB_TOKEN }}
    ```

2.  (Optional) Configure the analysis by creating a `.dependency-guardian.yml` file in your repository's root directory:

    ```yaml
    # .dependency-guardian.yml
    targets:
      high_level_packages:
        - "github.com/your-org/your-repo/app/*"
        - "github.com/your-org/your-repo/api/*"

    patterns:
      ignore_patterns:
        - "*_test.go"
        - "*/mocks/*"
    ```

## Configuration Examples

Here are a few examples to help you get started.

### Example 1: Focus on Application Entrypoints

This configuration is ideal if you want to monitor the final application binaries in your `cmd/` directory. The tool will only report on changes that have a downstream impact on these specific entrypoints.

```yaml
# .dependency-guardian.yml
targets:
  high_level_packages:
    # Only show impacts that affect packages in the cmd/ directory.
    - "**/cmd/**"

patterns:
  ignore_patterns:
    # A good default to reduce noise from test files.
    - "**/*_test.go"
```

### Example 2: Monitor Specific Business Logic

If your repository contains multiple services, you can configure the tool to watch for impacts on specific areas of business logic. This example focuses on the `billing` and `notifications` services.

```yaml
# .dependency-guardian.yml
targets:
  high_level_packages:
    - "**/services/billing/**"
    - "**/services/notifications/**"
```

### Example 3: Highlight Critical Security Packages

You can combine `targets` and `critical` to draw special attention to sensitive packages. This example ensures that any change affecting the `auth` package is not only reported but also flagged as critical.

```yaml
# .dependency-guardian.yml
targets:
  high_level_packages:
    - "**/pkg/auth/**"

critical:
  packages:
    - "**/pkg/auth/**"
```

## Development

Requirements:
- Go 1.24 or higher

Setup:
```bash
git clone https://github.com/cosmos/dependency-guardian.git
cd dependency-guardian
go mod tidy
```

Build:
```bash
go build -o dependency-guardian .
```

Run tests:
```bash
go test ./...
```

Run locally:
```bash
go run ./... analyze --owner <owner> --repo <repo> --pr <pr_number>
```

## License

MIT License 