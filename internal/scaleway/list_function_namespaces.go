package scaleway

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var listFunctionNamespacesTool = &mcp.Tool{
	Name:        "list_function_namespaces",
	Description: "List available Scaleway Function namespaces",
}

type ListFunctionNamespacesRequest struct{}

type ListFunctionNamespacesResponse struct {
	Namespaces []Namespace `json:"namespaces"`
}

func (t *Tools) ListFunctionNamespaces(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ ListFunctionNamespacesRequest,
) (*mcp.CallToolResult, ListFunctionNamespacesResponse, error) {
	resp, err := t.functionsAPI.ListNamespaces(
		&function.ListNamespacesRequest{},
		scw.WithContext(ctx),
	)
	if err != nil {
		return nil, ListFunctionNamespacesResponse{}, fmt.Errorf("listing namespaces: %w", err)
	}

	namespaces := make([]Namespace, 0, len(resp.Namespaces))

	for _, ns := range resp.Namespaces {
		namespaces = append(namespaces, NewNamespaceFromSDK(ns))
	}

	return nil, ListFunctionNamespacesResponse{Namespaces: namespaces}, nil
}
