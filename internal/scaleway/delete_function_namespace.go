package scaleway

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var deleteFunctionNamespaceTool = &mcp.Tool{
	Name:        "delete_function_namespace",
	Description: "Delete a Scaleway Function Namespace. It can only be used on namespaces created by this tool.",
}

type DeleteFunctionNamespaceRequest struct {
	NamespaceName string `json:"namespace_name"`
}

func (t *Tools) DeleteFunctionNamespace(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in DeleteFunctionNamespaceRequest,
) (*mcp.CallToolResult, Namespace, error) {
	ns, err := getFunctionNamespaceByName(ctx, t.functionsAPI, in.NamespaceName)
	if err != nil {
		return nil, Namespace{}, fmt.Errorf("getting namespace by name: %w", err)
	}

	if err := checkResourceOwnership(ns.Tags); err != nil {
		return nil, Namespace{}, err
	}

	ns, err = t.functionsAPI.DeleteNamespace(&function.DeleteNamespaceRequest{
		NamespaceID: ns.ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, Namespace{}, fmt.Errorf("deleting namespace: %w", err)
	}

	return nil, NewNamespaceFromSDK(ns), nil
}
