package github_test

import (
	"fmt"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"

	"github.com/softwaredevelop/pulumi-go-components/components/github"
)

// standardRepoMocks implements the pulumi.Mock interface for component testing.
type standardRepoMocks int

// NewResource provides a mock implementation for resource creation.
// This function is called for every resource creation.
// We must return mock data, especially for outputs that other resources
// or the component's outputs depend on.
func (m standardRepoMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := make(map[string]any)

	// Handle the creation of child resources within the component.
	switch args.TypeToken {
	case "custom:resource:StandardRepo":
		// The component resource itself doesn't need to mock any outputs.
		// Its outputs are constructed from its child resources.
	case "github:index/repository:Repository":
		// These outputs are used by the component's outputs.
		// It's crucial to mock them.
		repoName := args.Inputs["name"].StringValue()
		outputs["name"] = repoName
		outputs["htmlUrl"] = fmt.Sprintf("https://github.com/mock-owner/%s", repoName)
		outputs["nodeId"] = "mock-node-id-for-" + args.Name

	// For the other resources, we don't need to mock specific outputs
	// as the component does not directly depend on them. It's enough
	// that their creation succeeds without error.
	case "github:index/branchProtection:BranchProtection":
	case "github:index/issueLabel:IssueLabel":
	case "github:index/actionsSecret:ActionsSecret":

	default:
		return "", nil, fmt.Errorf("unknown resource type: %s", args.TypeToken)
	}

	// The physical ID can be a mock value.
	id := args.Name + "_id"
	return id, resource.NewPropertyMapFromMap(outputs), nil
}

// Call egy mock implementációt biztosít a függvény/provider hívásokhoz.
func (m standardRepoMocks) Call(_ pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

// assertOutputEquals is a helper function to reduce boilerplate in tests.
func assertOutputEquals[T any](t *testing.T, output pulumi.Output, expected T, msgAndArgs ...any) {
	t.Helper()
	output.ApplyT(func(v T) error {
		assert.Equal(t, expected, v, msgAndArgs...)
		return nil
	})
}

func TestNewStandardRepo(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		// Define the input arguments for the component.
		args := &github.StandardRepoArgs{
			RepositoryName: pulumi.String("test-repo"),
			Description:    pulumi.String("A test repository"),
			Topics:         pulumi.StringArray{pulumi.String("pulumi"), pulumi.String("go")},
		}

		// Create the component in the mock environment.
		repo, err := github.NewStandardRepo(ctx, "testStandardRepo", args)
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		// Verify that the component's outputs have the expected values.
		// The values are derived from the mocked Repository resource.
		assertOutputEquals(t, repo.RepositoryName, "test-repo", "RepositoryName should match the input")
		assertOutputEquals(t, repo.RepositoryURL, "https://github.com/mock-owner/test-repo", "RepositoryURL should be the mocked URL")
		// The child `repository` resource has the logical name "repository", so the mocked Node ID will contain it.
		assertOutputEquals(t, repo.RepositoryNodeID, "mock-node-id-for-repository", "RepositoryNodeID should be the mocked Node ID")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", standardRepoMocks(0)))

	assert.NoError(t, err)
}
