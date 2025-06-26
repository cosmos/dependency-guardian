# Dependency Impact Analyzer PRD

## Problem Statement

Developers frequently modify code without full awareness of which higher-level modules depend on their changes. This leads to insufficient testing and potential production issues, particularly in complex codebases with routing dependencies, asset routers (Axelar router, swap + CCTP router), and other critical components. Current test coverage is flaky, making manual QA and targeted testing essential, but developers lack visibility into what they should test.

## Solution Overview

Build a CLI tool that runs in GitHub Actions to analyze PRs and identify dependency tree changes, surfacing high-level modules that may be impacted by code modifications. The tool will use the `github.com/KyleBanks/depth` package to perform reverse dependency analysis and provide actionable insights to developers about potential testing areas through automated PR comments.

## Goals

### Primary Goals
- **Reduce production incidents** caused by insufficient testing of dependent modules
- **Improve developer awareness** of code change impact scope
- **Enable selective testing** by identifying which high-level components need validation
- **Integrate seamlessly** into existing GitHub PR workflow

### Secondary Goals
- Support integration with onchain test harness for expensive test selection
- Provide configurable filtering to reduce noise
- Enable team-specific customization of dependency tracking

## Success Metrics

- **30% reduction** in production incidents related to untested dependency changes
- **90% developer adoption** within 3 months of release
- **Average 2-minute analysis time** per PR
- **<5% false positive rate** for critical dependency alerts

## User Stories

### Developer Submitting PR
- As a developer, I want to see which high-level modules depend on my code changes so I can test the right components
- As a developer, I want to understand if my routing dependency changes affect specific routers so I can validate them manually
- As a developer, I want the analysis to run automatically in CI so I don't have to remember to trigger it

### Code Reviewer
- As a reviewer, I want to see dependency impact analysis in PRs so I can suggest appropriate testing
- As a reviewer, I want to understand the blast radius of changes so I can assess risk appropriately

### Engineering Manager
- As an EM, I want to configure which packages are tracked so we focus on business-critical dependencies
- As an EM, I want to ignore noisy dependencies so developers get actionable insights

## Functional Requirements

### Core Analysis Engine
- **FR-1**: Analyze modified files in GitHub PRs using the `github.com/KyleBanks/depth` package (maybe fork it and reuse the code)
- **FR-2**: Generate reverse dependency graphs showing which high-level modules depend on changed code
- **FR-3**: Support Go modules and package analysis (primary target)
- **FR-4**: Process incremental changes (not full codebase) for performance

### Configuration System
- **FR-5**: Allow targeting specific "high-level" packages for analysis (e.g., routers, services)
- **FR-6**: Support include/exclude patterns for packages to reduce noise
- **FR-7**: Enable team-specific configuration files (`.dependency-guardian.yml`)
- **FR-8**: Support regex patterns for flexible package matching

### GitHub Integration
- **FR-9**: Post analysis results as PR comments
- **FR-10**: Update comments on subsequent pushes rather than creating new ones
- **FR-11**: Support GitHub Actions workflow integration
- **FR-12**: Provide status checks for critical dependency changes

### Output Format
- **FR-13**: Generate hierarchical dependency trees showing impact paths
- **FR-14**: Highlight critical dependencies (configurable)
- **FR-15**: Provide actionable testing recommendations
- **FR-16**: Link to relevant test files/commands when available

## Technical Specifications

### Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  GitHub Actions │───▶│   CLI Tool      │───▶│  depth Package  │
│     Runner      │    │                 │    │   Analysis      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │  Impact Report  │
                       │   Generator     │
                       └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   GitHub API    │
                       │  Comment Post   │
                       └─────────────────┘
```

### Configuration Schema
```yaml
# .dependency-guardian.yml
targets:
  high_level_packages:
    - "github.com/company/project/routers/*"
    - "github.com/company/project/services/*"
  
ignore_patterns:
    - "*/testutil/*"
    - "*/mocks/*"
    - "*/proto/*"
  
include_patterns:
    - "github.com/company/project/*"
  
critical_packages:
    - "github.com/company/project/routers/axelar"
    - "github.com/company/project/routers/swap-cctp"
  
analysis:
  max_depth: 5
  min_impact_threshold: 2
```

### Output Example
```markdown
## 🔍 Dependency Impact Analysis

### High-Level Modules Affected:
- **🚨 Critical**: `routers/axelar` (via `core/routing` → `util/validation`)
- **⚠️ Medium**: `services/bridge` (via `core/routing`)

### Recommended Testing:
- [ ] Test Axelar router integration flows
- [ ] Validate CCTP routing functionality  
- [ ] Run bridge service smoke tests

### Dependency Paths:
```
routers/axelar
  └── core/routing (modified)
      └── util/validation (modified)

services/bridge  
  └── core/routing (modified)
```

<details>
<summary>Full Analysis Details</summary>
[Detailed dependency tree...]
</details>
```

## Non-Functional Requirements

### Performance
- **NFR-1**: Analysis completes within 2 minutes for typical PRs (<50 files)
- **NFR-2**: Support repositories up to 10k Go files
- **NFR-3**: Graceful degradation for very large PRs (>100 files)

### Reliability  
- **NFR-4**: 99.5% uptime for GitHub Actions integration
- **NFR-5**: Graceful failure with helpful error messages
- **NFR-6**: Retry logic for transient GitHub API failures

### Usability
- **NFR-7**: Zero configuration required for basic usage
- **NFR-8**: Clear, actionable output format
- **NFR-9**: Mobile-friendly GitHub comment formatting

## Implementation Phases

### Phase 1: Core CLI Tool (4 weeks)
- Integrate `depth` package for dependency analysis
- Build basic reverse dependency graph generation
- Create configuration parsing system
- Implement file change detection from GitHub PR diffs
- Build CLI interface with GitHub Actions integration

### Phase 2: GitHub Integration (3 weeks)  
- Implement PR comment posting/updating via GitHub API
- Create reusable GitHub Actions workflow
- Add status check integration
- Build environment variable configuration for CI

### Phase 3: Advanced Features (3 weeks)
- Add critical dependency highlighting
- Implement testing recommendation engine
- Build configuration validation
- Add support for monorepo detection

### Phase 4: Polish & Distribution (2 weeks)
- Performance optimization
- Error handling improvements  
- Documentation and GitHub Actions marketplace publishing
- Beta testing with select repositories

## Risks and Mitigations

### Technical Risks
- **Risk**: `depth` package performance on large codebases
  - **Mitigation**: Implement caching and incremental analysis
- **Risk**: GitHub API rate limiting
  - **Mitigation**: Implement smart caching and batching

### Product Risks  
- **Risk**: Too many false positives leading to alert fatigue
  - **Mitigation**: Careful default configuration and user feedback loops
- **Risk**: Analysis too slow for developer workflow
  - **Mitigation**: Async analysis with quick previews

## Success Criteria

### Launch Criteria
- [ ] Successfully analyzes PRs in <2 minutes 90% of the time
- [ ] Generates actionable recommendations for 80% of dependency changes
- [ ] Zero critical bugs in GitHub integration
- [ ] Documentation and onboarding materials complete

### 6-Month Success Metrics
- 30% reduction in production incidents from dependency changes
- 90% developer adoption rate
- <5% false positive rate for critical alerts
- Average developer satisfaction score >4.0/5.0

## Future Considerations

- **Multi-language support**: Extend beyond Go to TypeScript, Python
- **IDE integration**: VS Code extension for local analysis
- **ML-powered recommendations**: Learn from historical incident patterns
- **Test execution integration**: Automatically trigger relevant test suites
- **Slack/Teams notifications**: Alert relevant team channels for critical changes
- **GitHub Actions marketplace**: Publish as reusable action for broader community