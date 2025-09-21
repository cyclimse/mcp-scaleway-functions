package scaleway

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var createAndDeployFunctionNamespaceTool = &mcp.Tool{
	Name:        "create_and_deploy_function_namespace",
	Description: "Create and deploy a Scaleway Function Namespace",
}

// We could embed function.CreateNamespaceRequest but:
// - The LLM seems to be confused about the `project_id` field which we don't need
// to set and instead rely on the provider default project.
type CreateAndDeployFunctionNamespace struct {
	Name string   `json:"name"`
	Tags []string `json:"tags,omitempty"`
}

func (in CreateAndDeployFunctionNamespace) ToSDK() *function.CreateNamespaceRequest {
	return &function.CreateNamespaceRequest{
		Name: in.Name,
		Tags: setCreatedByTagIfAbsent(in.Tags),
	}
}

func (t *Tools) CreateAndDeployFunctionNamespace(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in CreateAndDeployFunctionNamespace,
) (*mcp.CallToolResult, Namespace, error) {
	ns, err := t.functionsAPI.CreateNamespace(in.ToSDK(), scw.WithContext(ctx))
	if err != nil {
		return nil, Namespace{}, fmt.Errorf("creating namespace: %w", err)
	}

	ns, err = t.functionsAPI.WaitForNamespace(&function.WaitForNamespaceRequest{
		NamespaceID: ns.ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, Namespace{}, fmt.Errorf("waiting for namespace to be ready: %w", err)
	}

	return nil, NewNamespaceFromSDK(ns), nil
}
