package scaleway

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var listFunctionRuntimesTool = &mcp.Tool{
	Name:        "list_function_runtimes",
	Description: "List available Scaleway Function runtimes",
}

type ListFunctionRuntimesRequest struct{}

type ListFunctionRuntimesResponse struct {
	Runtimes []Runtime `json:"runtimes"`
}

func (t *Tools) ListFunctionRuntimes(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ ListFunctionRuntimesRequest,
) (*mcp.CallToolResult, ListFunctionRuntimesResponse, error) {
	resp, err := t.functionsAPI.ListFunctionRuntimes(
		&function.ListFunctionRuntimesRequest{},
		scw.WithContext(ctx),
	)
	if err != nil {
		return nil, ListFunctionRuntimesResponse{}, fmt.Errorf("listing runtimes: %w", err)
	}

	runtimes := make([]Runtime, 0, len(resp.Runtimes))

	for _, r := range resp.Runtimes {
		runtimes = append(runtimes, NewRuntimeFromSDK(r))
	}

	return nil, ListFunctionRuntimesResponse{Runtimes: runtimes}, nil
}
