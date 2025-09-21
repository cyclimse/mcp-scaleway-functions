package scaleway

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var deleteFunctionTool = &mcp.Tool{
	Name:        "delete_function",
	Description: "Delete a Scaleway Function. It can only be used on functions created by this tool.",
}

type DeleteFunctionRequest struct {
	FunctionName string `json:"function_name"`
}

func (t *Tools) DeleteFunction(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in DeleteFunctionRequest,
) (*mcp.CallToolResult, Function, error) {
	fun, err := getFunctionByName(ctx, t.functionsAPI, in.FunctionName)
	if err != nil {
		return nil, Function{}, fmt.Errorf("getting function by name: %w", err)
	}

	if err := checkResourceOwnership(fun.Tags); err != nil {
		return nil, Function{}, err
	}

	fun, err = t.functionsAPI.DeleteFunction(&function.DeleteFunctionRequest{
		FunctionID: fun.ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, Function{}, fmt.Errorf("deleting function: %w", err)
	}

	return nil, NewFunctionFromSDK(fun), nil
}
