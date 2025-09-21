package scaleway

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var listFunctionsTool = &mcp.Tool{
	Name:        "list_functions",
	Description: "List Scaleway Functions",
}

type ListFunctionsRequest struct {
	function.ListFunctionsRequest
}

type ListFunctionsResponse struct {
	Functions []Function `json:"functions"`
}

func (t *Tools) ListFunctions(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in ListFunctionsRequest,
) (*mcp.CallToolResult, ListFunctionsResponse, error) {
	resp, err := t.functionsAPI.ListFunctions(
		&in.ListFunctionsRequest,
		scw.WithAllPages(),
		scw.WithContext(ctx),
	)
	if err != nil {
		return nil, ListFunctionsResponse{}, fmt.Errorf("listing functions: %w", err)
	}

	functions := make([]Function, 0, len(resp.Functions))

	for _, f := range resp.Functions {
		functions = append(functions, NewFunctionFromSDK(f))
	}

	return nil, ListFunctionsResponse{Functions: functions}, nil
}
