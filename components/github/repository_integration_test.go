//go:build integration
// +build integration

package github_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	// Package for the component to be tested.
	// The module name must be the full, version-controllable path.
	github "github.com/softwaredevelop/pulumi-go-components/components/github"
)

func TestStandardRepo_Integration(t *testing.T) {
	// This test creates real resources, so it should only run when
	// the necessary environment variables are set.
	pulumiAccessToken := os.Getenv("PULUMI_ACCESS_TOKEN")
	pulumiOrgName := os.Getenv("PULUMI_ORG_NAME")
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubOwner := os.Getenv("GITHUB_OWNER")

	require.NotEmpty(t, pulumiAccessToken, "PULUMI_ACCESS_TOKEN environment variable must be set for integration tests")
	require.NotEmpty(t, pulumiOrgName, "PULUMI_ORG_NAME environment variable must be set for integration tests")
	require.NotEmpty(t, githubToken, "GITHUB_TOKEN environment variable must be set for integration tests")
	require.NotEmpty(t, githubOwner, "GITHUB_OWNER environment variable must be set for integration tests")

	ctx := context.Background()

	projectName := "component-test"
	repoName := fmt.Sprintf("test-repo-%d", time.Now().Unix())
	stackName := auto.FullyQualifiedStackName(pulumiOrgName, projectName, repoName)

	// Define the Pulumi program inline, within the test.
	// This program will use the StandardRepo component.
	program := func(pCtx *pulumi.Context) error {
		repo, err := github.NewStandardRepo(pCtx, "my-standard-repo", &github.StandardRepoArgs{
			RepositoryName: pulumi.String(repoName),
			Description:    pulumi.String("Temporary repository for integration testing"),
			Topics:         pulumi.StringArray{pulumi.String("pulumi"), pulumi.String("testing")},
		})
		if err != nil {
			return fmt.Errorf("failed to create StandardRepo component: %w", err)
		}

		// Export the component's outputs so we can verify them.
		pCtx.Export("repositoryName", repo.RepositoryName)
		pCtx.Export("repositoryUrl", repo.RepositoryURL)
		pCtx.Export("repositoryNodeId", repo.RepositoryNodeID)

		return nil
	}

	// Create the stack with the inline program.
	stack, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, program)
	require.NoError(t, err, "Failed to create stack")

	// t.Cleanup ensures that the registered functions run at the end of the test,
	// (both on success and failure) to clean up the resources.
	t.Cleanup(func() {
		t.Log("Destroying integration test stack...")
		_, destroyErr := stack.Destroy(ctx)
		assert.NoError(t, destroyErr, "Stack destruction should not fail")

		ws, err := auto.NewLocalWorkspace(ctx)
		if assert.NoError(t, err, "Failed to get workspace for cleanup") {
			err = ws.RemoveStack(ctx, stackName)
			assert.NoError(t, err, "Stack removal should not fail")
		}
		t.Log("Integration test stack and its resources have been destroyed.")
	})

	// Set the required configuration for the GitHub provider.
	err = stack.SetAllConfig(ctx, auto.ConfigMap{
		"github:token": auto.ConfigValue{Value: githubToken, Secret: true},
		"github:owner": auto.ConfigValue{Value: githubOwner, Secret: true},
	})
	require.NoError(t, err, "Failed to set configuration")

	// Run the `pulumi up` command on the stack.
	upRes, err := stack.Up(ctx)
	require.NoError(t, err, "stack.Up execution failed")

	// Check the results.
	assert.NotNil(t, upRes.Outputs, "Outputs should not be nil")

	nameOutput, ok := upRes.Outputs["repositoryName"]
	assert.True(t, ok, "The 'repositoryName' output must exist")
	assert.Equal(t, repoName, nameOutput.Value)

	urlOutput, ok := upRes.Outputs["repositoryUrl"]
	assert.True(t, ok, "The 'repositoryUrl' output must exist")
	expectedURL := fmt.Sprintf("https://github.com/%s/%s", githubOwner, repoName)
	assert.Equal(t, expectedURL, urlOutput.Value)

	nodeIDOutput, ok := upRes.Outputs["repositoryNodeId"]
	assert.True(t, ok, "The 'repositoryNodeId' output must exist")
	assert.NotEmpty(t, nodeIDOutput.Value, "The 'repositoryNodeId' should not be empty")
}
