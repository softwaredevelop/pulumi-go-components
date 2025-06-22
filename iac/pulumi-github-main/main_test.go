//revive:disable:package-comments,exported
package main

import (
	"testing"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
)

// mocks implements the pulumi.Mock interface for component testing.
type mocks int

// NewResource provides a mock implementation of resource creation.
func (m mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	// This function is called for every resource being created.
	// We must return mock data, especially for outputs that other resources depend on.
	outputs := make(map[string]any)

	// Populate outputs based on the resource type and inputs.
	switch args.TypeToken {
	case "github:index/repository:Repository":
		// These outputs are used by other resources (e.g., BranchProtection, IssueLabel).
		outputs["name"] = args.Inputs["name"]
		outputs["nodeId"] = pulumi.ID("mock-node-id-for-" + args.Name)
		// We can also mock other properties we want to test.
		outputs["visibility"] = args.Inputs["visibility"]
		outputs["deleteBranchOnMerge"] = args.Inputs["deleteBranchOnMerge"]
	case "github:index/branchProtection:BranchProtection":
		outputs["pattern"] = args.Inputs["pattern"]
		outputs["requiredLinearHistory"] = args.Inputs["requiredLinearHistory"]
	case "github:index/issueLabel:IssueLabel":
		outputs["name"] = args.Inputs["name"]
		outputs["color"] = args.Inputs["color"]
		outputs["description"] = args.Inputs["description"]
		outputs["repository"] = args.Inputs["repository"]
	case "github:index/actionsSecret:ActionsSecret":
		outputs["repository"] = args.Inputs["repository"]
		outputs["secretName"] = args.Inputs["secretName"]
	}

	// The physical ID can be a mock value.
	id := args.Name + "_id"
	return id, resource.NewPropertyMapFromMap(outputs), nil
}

// Call provides a mock implementation of function/provider calls.
func (m mocks) Call(_ pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

// assertOutputEquals is a helper function to reduce boilerplate in tests.
// It applies an assertion to a Pulumi output.
func assertOutputEquals[T any](t *testing.T, output pulumi.Output, expected T, msgAndArgs ...any) {
	t.Helper()
	output.ApplyT(func(v T) error {
		assert.Equal(t, expected, v, msgAndArgs...)
		return nil
	})
}

func TestDefineInfrastructure(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		resources, err := defineInfrastructure(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, resources)

		t.Run("Repository", func(t *testing.T) {
			assertOutputEquals(t, resources.Repository.Name, "pulumi-go-components", "Repository name should match")
			assertOutputEquals(t, resources.Repository.Visibility, "public", "Repository visibility should be public")
			// The DeleteBranchOnMerge property is a pointer type (*bool), so we handle it with a specific ApplyT.
			resources.Repository.DeleteBranchOnMerge.ApplyT(func(v *bool) error {
				assert.NotNil(t, v, "DeleteBranchOnMerge should have a value")
				if v != nil {
					assert.True(t, *v, "DeleteBranchOnMerge should be true")
				}
				return nil
			})
		})

		t.Run("BranchProtection", func(t *testing.T) {
			resources.BranchProtection.Pattern.ApplyT(func(pattern string) error {
				assert.Equal(t, "main", pattern, "Branch protection pattern should be 'main'")
				return nil
			})
		})

		labelTests := []struct {
			name          string
			label         *github.IssueLabel
			expectedName  string
			expectedColor string
			expectedDesc  string
		}{
			{"GhActionsLabel", resources.GhActionsLabel, "github-actions dependencies", "E66E01", "This issue is related to github-actions dependencies"},
			{"GoModulesLabel", resources.GoModulesLabel, "go-modules dependencies", "9BE688", "This issue is related to go modules dependencies"},
		}

		for _, tt := range labelTests {
			t.Run("IssueLabel/"+tt.name, func(t *testing.T) {
				assertOutputEquals(t, tt.label.Name, tt.expectedName)
				assertOutputEquals(t, tt.label.Color, tt.expectedColor)
				// The Description property is optional and thus a pointer type (*string).
				// We handle it with a specific ApplyT to correctly dereference the pointer.
				tt.label.Description.ApplyT(func(v *string) error {
					assert.NotNil(t, v, "Description should have a value")
					if v != nil {
						assert.Equal(t, tt.expectedDesc, *v)
					}
					return nil
				})
				pulumi.All(resources.Repository.Name, tt.label.Repository).ApplyT(func(args []any) error {
					repoName := args[0].(string)
					labelRepo := args[1].(string)
					assert.Equal(t, repoName, labelRepo, "should be in the correct repository")
					return nil
				})
			})
		}

		secretTests := []struct {
			name         string
			secret       *github.ActionsSecret
			expectedName string
		}{
			{"GitlabRepoSecret", resources.GitlabRepoSecret, "GITLAB_REPOSITORY"},
			{"GitlabTokenSecret", resources.GitlabTokenSecret, "GITLAB_TOKEN"},
			{"GitlabOwnerSecret", resources.GitlabOwnerSecret, "GITLAB_OWNER"},
		}

		for _, tt := range secretTests {
			t.Run("ActionsSecret/"+tt.name, func(t *testing.T) {
				assertOutputEquals(t, tt.secret.SecretName, tt.expectedName)
				pulumi.All(resources.Repository.Name, tt.secret.Repository).ApplyT(func(args []any) error {
					repoName := args[0].(string)
					secretRepo := args[1].(string)
					assert.Equal(t, repoName, secretRepo, "should be in the correct repository")
					return nil
				})
			})
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", mocks(0)))

	assert.NoError(t, err)
}
