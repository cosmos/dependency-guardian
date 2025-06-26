package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client with our custom functionality
type Client struct {
	client *github.Client
	ctx    context.Context
}

// NewClient creates a new GitHub client using the GITHUB_TOKEN environment variable
func NewClient() (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &Client{
		client: client,
		ctx:    ctx,
	}, nil
}

// GetPullRequest fetches a pull request by number
func (c *Client) GetPullRequest(owner, repo string, number int) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(c.ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR #%d: %w", number, err)
	}
	return pr, nil
}

// GetPullRequestFiles fetches all files changed in a pull request, handling pagination
func (c *Client) GetPullRequestFiles(owner, repo string, number int) ([]*github.CommitFile, error) {
	var allFiles []*github.CommitFile
	opts := &github.ListOptions{
		PerPage: 100, // Maximum allowed by GitHub API
	}

	for {
		files, resp, err := c.client.PullRequests.ListFiles(c.ctx, owner, repo, number, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch PR #%d files: %w", number, err)
		}

		allFiles = append(allFiles, files...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allFiles, nil
}

// ListComments lists all comments on a pull request
func (c *Client) ListComments(owner, repo string, number int) ([]*github.IssueComment, error) {
	comments, _, err := c.client.Issues.ListComments(c.ctx, owner, repo, number, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments on PR #%d: %w", number, err)
	}
	return comments, nil
}

// UpdateComment updates an existing comment on a pull request
func (c *Client) UpdateComment(owner, repo string, commentID int64, body string) error {
	comment := &github.IssueComment{Body: &body}
	_, _, err := c.client.Issues.EditComment(c.ctx, owner, repo, commentID, comment)
	if err != nil {
		return fmt.Errorf("failed to update comment #%d: %w", commentID, err)
	}
	return nil
}

// CreateComment creates a new comment on a pull request
func (c *Client) CreateComment(owner, repo string, number int, body string) error {
	comment := &github.IssueComment{Body: &body}
	_, _, err := c.client.Issues.CreateComment(c.ctx, owner, repo, number, comment)
	if err != nil {
		return fmt.Errorf("failed to create comment on PR #%d: %w", number, err)
	}
	return nil
} 