//revive:disable:package-comments,exported
package main

import (
	"context"
	"errors"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/stretchr/testify/assert"
)

// mockStack is a mock implementation of the Stack interface for testing.
type mockStack struct {
	SetEnvVarsErr   error
	SetAllConfigErr error
	RefreshErr      error
	PreviewErr      error
	UpResult        auto.UpResult
	UpErr           error
	DestroyErr      error
	RefreshOut      string
	PreviewOut      string
}

func (m *mockStack) SetEnvVars(_ map[string]string) error {
	return m.SetEnvVarsErr
}

func (m *mockStack) SetAllConfig(_ context.Context, _ auto.ConfigMap) error {
	return m.SetAllConfigErr
}

func (m *mockStack) Refresh(_ context.Context) (string, error) {
	return m.RefreshOut, m.RefreshErr
}

func (m *mockStack) Preview(_ context.Context) (string, error) {
	return m.PreviewOut, m.PreviewErr
}

func (m *mockStack) Up(_ context.Context) (auto.UpResult, error) {
	return m.UpResult, m.UpErr
}

func (m *mockStack) Destroy(_ context.Context) error {
	return m.DestroyErr
}

func TestDeployStack(t *testing.T) {
	ctx := context.Background()
	configMap := auto.ConfigMap{}
	accessToken := "fake-token"

	tests := []struct {
		name        string
		mock        *mockStack
		expectErr   bool
		expectedMsg string
	}{
		{
			name: "Success",
			mock: &mockStack{
				RefreshOut: "refresh success",
				PreviewOut: "preview success",
				UpResult: auto.UpResult{
					StdOut:  "up success",
					Outputs: map[string]auto.OutputValue{"repositoryUrl": {Value: "https://github.com/test/repo"}},
				},
			},
			expectErr: false,
		},
		{
			name:        "SetEnvVars fails",
			mock:        &mockStack{SetEnvVarsErr: errors.New("set env failed")},
			expectErr:   true,
			expectedMsg: "failed to set environment variables: set env failed",
		},
		{
			name:        "SetAllConfig fails",
			mock:        &mockStack{SetAllConfigErr: errors.New("set config failed")},
			expectErr:   true,
			expectedMsg: "failed to set config: set config failed",
		},
		{
			name:        "Refresh fails",
			mock:        &mockStack{RefreshErr: errors.New("refresh failed")},
			expectErr:   true,
			expectedMsg: "failed to refresh stack: refresh failed",
		},
		{
			name:        "Preview fails",
			mock:        &mockStack{PreviewErr: errors.New("preview failed")},
			expectErr:   true,
			expectedMsg: "failed to preview stack: preview failed",
		},
		{
			name:        "Up fails",
			mock:        &mockStack{UpErr: errors.New("up failed")},
			expectErr:   true,
			expectedMsg: "failed to update stack: up failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputs, err := deployStack(ctx, tt.mock, accessToken, configMap)

			if tt.expectErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedMsg)
				assert.Nil(t, outputs)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, outputs)
				assert.Equal(t, "https://github.com/test/repo", outputs["repositoryUrl"].Value)
			}
		})
	}
}
