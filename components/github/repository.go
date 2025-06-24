// Package github provides a Pulumi component for creating a standardized GitHub repository.
package github

import (
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// StandardRepoArgs defines the input parameters for our component.
// Anything that the user needs to be able to customize should be placed here.
type StandardRepoArgs struct {
	// The name of the repository to be created on GitHub.
	RepositoryName pulumi.StringInput
	// The description of the repository.
	Description pulumi.StringInput
	// The topics to be assigned to the repository.
	Topics pulumi.StringArrayInput
}

// StandardRepo is our custom component.
// The output properties (e.g., URL) are defined here with `pulumi:"..."` tags.
type StandardRepo struct {
	pulumi.ResourceState

	// Output properties that we want to access after using the component.
	RepositoryName   pulumi.StringOutput `pulumi:"repositoryName"`
	RepositoryURL    pulumi.StringOutput `pulumi:"repositoryUrl"`
	RepositoryNodeID pulumi.StringOutput `pulumi:"repositoryNodeId"`

	// Expose the underlying repository resource to allow for composition.
	Repository *github.Repository `pulumi:"repository"`
}

// NewStandardRepo is the constructor function for our component.
// It creates the component and the "child" resources within it.
func NewStandardRepo(ctx *pulumi.Context, name string, args *StandardRepoArgs, opts ...pulumi.ResourceOption) (*StandardRepo, error) {
	// STEP 1: Register the component with the Pulumi engine.
	// The first argument is a unique type name for Pulumi.
	standardRepo := &StandardRepo{}
	err := ctx.RegisterComponentResource("custom:resource:StandardRepo", name, standardRepo, opts...)
	if err != nil {
		return nil, err
	}

	// STEP 2: Create a "parent" option. This ensures that all
	// created resources are logically part of our component.
	parentOpt := pulumi.Parent(standardRepo)

	// STEP 3: The full logic of `defineInfrastructure` is copied here,
	// and the hardcoded values are replaced with those from `args`.

	repository, err := github.NewRepository(ctx, "repository", &github.RepositoryArgs{
		Name:                args.RepositoryName,
		Description:         args.Description,
		Topics:              args.Topics,
		DeleteBranchOnMerge: pulumi.Bool(true),
		HasIssues:           pulumi.Bool(true),
		HasProjects:         pulumi.Bool(true),
		Visibility:          pulumi.String("public"),
	}, parentOpt) // Important: the component is the parent!
	if err != nil {
		return nil, err
	}

	_, err = github.NewBranchProtection(ctx, "branch-protection", &github.BranchProtectionArgs{
		RepositoryId:          repository.NodeId,
		Pattern:               pulumi.String("main"),
		RequiredLinearHistory: pulumi.Bool(true),
	}, parentOpt) // Important: the component is the parent!
	if err != nil {
		return nil, err
	}

	_, err = github.NewIssueLabel(ctx, "label-gh-actions", &github.IssueLabelArgs{
		Repository:  repository.Name,
		Name:        pulumi.String("github-actions dependencies"),
		Color:       pulumi.String("E66E01"),
		Description: pulumi.String("This issue is related to github-actions dependencies"),
	}, parentOpt) // Important: the component is the parent!
	if err != nil {
		return nil, err
	}

	// The Parent was already correctly set for the secrets, but now it too
	// must point to the component, not directly to the repository.
	_, err = github.NewActionsSecret(ctx, "secret-gitlab-repo", &github.ActionsSecretArgs{
		Repository: repository.Name,
		SecretName: pulumi.String("GITLAB_REPOSITORY"),
	}, parentOpt)
	if err != nil {
		return nil, err
	}

	_, err = github.NewActionsSecret(ctx, "secret-gitlab-token", &github.ActionsSecretArgs{
		Repository: repository.Name,
		SecretName: pulumi.String("GITLAB_TOKEN"),
	}, parentOpt)
	if err != nil {
		return nil, err
	}

	_, err = github.NewActionsSecret(ctx, "secret-gitlab-owner", &github.ActionsSecretArgs{
		Repository: repository.Name,
		SecretName: pulumi.String("GITLAB_OWNER"),
	}, parentOpt)
	if err != nil {
		return nil, err
	}

	// STEP 4: Set the output properties of the component.
	standardRepo.RepositoryName = repository.Name
	standardRepo.RepositoryURL = repository.HtmlUrl
	standardRepo.RepositoryNodeID = repository.NodeId
	standardRepo.Repository = repository

	// STEP 5: Register the outputs so the Pulumi engine can see them.
	if err := ctx.RegisterResourceOutputs(standardRepo, pulumi.Map{
		"repositoryName":   standardRepo.RepositoryName,
		"repositoryUrl":    standardRepo.RepositoryURL,
		"repositoryNodeId": standardRepo.RepositoryNodeID,
		"repository":       standardRepo.Repository,
	}); err != nil {
		return nil, err
	}

	return standardRepo, nil
}
