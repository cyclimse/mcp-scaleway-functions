package scaleway

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var downloadFunctionTool = &mcp.Tool{
	Name: "download_function",
	Description: `Download the code of a Scaleway Function.
	The provided "to_directory" must be an existing directory where the function code will be extracted.`,
}

type DownloadFunctionRequest struct {
	FunctionName string `json:"function_name"`
	ToDirectory  string `json:"to_directory"`
}

func (t *Tools) DownloadFunction(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in DownloadFunctionRequest,
) (*mcp.CallToolResult, Function, error) {
	fun, err := getFunctionByName(ctx, t.functionsAPI, in.FunctionName)
	if err != nil {
		return nil, Function{}, fmt.Errorf("getting function by name: %w", err)
	}

	url, err := t.functionsAPI.GetFunctionDownloadURL(&function.GetFunctionDownloadURLRequest{
		FunctionID: fun.ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, Function{}, fmt.Errorf("getting function download URL: %w", err)
	}

	if err := DownloadAndExtractCodeArchive(ctx, url.URL, in.ToDirectory); err != nil {
		return nil, Function{}, fmt.Errorf("downloading and extracting function: %w", err)
	}

	return nil, NewFunctionFromSDK(fun), nil
}
