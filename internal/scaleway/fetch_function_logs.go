package scaleway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	"github.com/cyclimse/mcp-scaleway-functions/internal/scaleway/cockpit"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//nolint:gochecknoglobals
var fetchFunctionLogsTool = &mcp.Tool{
	Name:        "fetch_function_logs",
	Description: "Fetch logs for a specific Scaleway Function",
}

type FetchFunctionLogsRequest struct {
	FunctionName string    `json:"function_name"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
}

type FetchFunctionLogsResponse struct {
	Logs []cockpit.Log `json:"logs"`
}

func (t *Tools) FetchFunctionLogs(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	req FetchFunctionLogsRequest,
) (*mcp.CallToolResult, FetchFunctionLogsResponse, error) {
	function, ns, err := getFunctionAndNamespaceByFunctionName(ctx, t.functionsAPI, req.FunctionName)
	if err != nil {
		return nil, FetchFunctionLogsResponse{}, fmt.Errorf("getting function by name: %w", err)
	}

	if ns.ProjectID != t.projectID {
		errMsg := "Currently %s does not support fetching logs in multiple Scaleway projects.\n" +
			"Make sure to use a Scaleway profile with SCW_DEFAULT_PROJECT_ID set to %q.\n" +
			"Please open an issue on GitHub if you need this feature."
		return nil, FetchFunctionLogsResponse{}, fmt.Errorf(errMsg, constants.ProjectName, ns.ProjectID)
	}

	// The resource name used in Cockpit Logs is the function's subdomain (the part before the first dot).
	resourceName, _, _ := strings.Cut(function.DomainName, ".")

	logs, err := t.cockpitClient.ListFunctionLogs(ctx, resourceName, req.StartTime, req.EndTime)
	if err != nil {
		return nil, FetchFunctionLogsResponse{}, fmt.Errorf("listing function logs: %w", err)
	}

	return nil, FetchFunctionLogsResponse{Logs: logs}, nil
}
