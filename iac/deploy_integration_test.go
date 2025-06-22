//go:build integration
// +build integration

//revive:disable:package-comments,exported
package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployStack_Integration(t *testing.T) {
	// This test creates real resources, so it should only run when
	// the necessary environment variables are set.
	pulumiAccessToken := os.Getenv("PULUMI_ACCESS_TOKEN")
	pulumiOrgName := os.Getenv("PULUMI_ORG_NAME")
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubOwner := os.Getenv("GITHUB_OWNER")

	require.NotEmpty(t, pulumiAccessToken, "PULUMI_ACCESS_TOKEN must be set for integration tests")
	require.NotEmpty(t, pulumiOrgName, "PULUMI_ORG_NAME must be set for integration tests")
	require.NotEmpty(t, githubToken, "GITHUB_TOKEN must be set for integration tests")
	require.NotEmpty(t, githubOwner, "GITHUB_OWNER must be set for integration tests")

	ctx := context.Background()

	// Use a unique stack name for each test run to avoid collisions.
	stackProjectName := "components"
	stackEnvironmentName := fmt.Sprintf("test-%d", time.Now().Unix())
	workDir := "pulumi-github-main"
	pulumiStackName := auto.FullyQualifiedStackName(pulumiOrgName, stackProjectName, stackEnvironmentName)

	// Use the real pulumiStack implementation, not the mock.
	stack, err := NewPulumiStack(ctx, pulumiStackName, workDir)
	require.NoError(t, err, "Failed to create a real stack for testing. This can happen if the workDir is incorrect.")

	// t.Cleanup ensures that the registered functions will run at the end of the test,
	// whether it passes or fails, to perform cleanup.
	t.Cleanup(func() {
		t.Log("Destroying integration test stack...")
		destroyErr := stack.Destroy(ctx)
		assert.NoError(t, destroyErr, "Stack destruction should not fail")

		// A stack-et magát is eltávolítjuk a workspace-ből.
		ws, err := auto.NewLocalWorkspace(ctx)
		if assert.NoError(t, err, "Failed to get workspace for cleanup") {
			err = ws.RemoveStack(ctx, pulumiStackName)
			assert.NoError(t, err, "Stack removal should not fail")
		}
		t.Log("Integration test stack and resources destroyed.")
	})

	configMap := auto.ConfigMap{
		"github:token": auto.ConfigValue{
			Value:  githubToken,
			Secret: true,
		},
		"github:owner": auto.ConfigValue{
			Value:  githubOwner,
			Secret: true,
		},
	}

	// Run the function under test with the real stack.
	outputs, err := deployStack(ctx, stack, pulumiAccessToken, configMap)

	// Assert the results.
	assert.NoError(t, err, "deployStack should complete without error in integration test")
	assert.NotNil(t, outputs, "Outputs should not be nil on success")
	assert.Contains(t, outputs, "repositoryUrl", "Outputs should contain repositoryUrl")
	assert.NotEmpty(t, outputs["repositoryUrl"].Value, "repositoryUrl should have a value")
	assert.Contains(t, outputs["repositoryUrl"].Value, "https://github.com/", "repositoryUrl should be a valid GitHub URL")
}
