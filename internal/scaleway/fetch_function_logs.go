package scaleway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cyclimse/mcp-scaleway-functions/internal/scaleway/cockpit"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var ErrMultipleProjectsNotSupported = errors.New(
	"fetching logs across multiple Scaleway projects is not supported yet",
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
	function, ns, err := getFunctionAndNamespaceByFunctionName(
		ctx,
		t.functionsAPI,
		req.FunctionName,
	)
	if err != nil {
		return nil, FetchFunctionLogsResponse{}, fmt.Errorf("getting function by name: %w", err)
	}

	if ns.ProjectID != t.projectID {
		slog.WarnContext(
			ctx,
			"fetching logs across multiple Scaleway projects is not supported yet",
			"function_project_id", ns.ProjectID,
			"active_profile_project_id", t.projectID,
		)

		return nil, FetchFunctionLogsResponse{}, fmt.Errorf(
			"%w: function %q is in project %q, but the active Scaleway profile is in project %q",
			ErrMultipleProjectsNotSupported,
			req.FunctionName,
			ns.ProjectID,
			t.projectID,
		)
	}

	// The resource name used in Cockpit Logs is the function's subdomain (the part before the first dot).
	resourceName, _, _ := strings.Cut(function.DomainName, ".")

	logs, err := t.cockpitClient.ListFunctionLogs(ctx, resourceName, req.StartTime, req.EndTime)
	if err != nil {
		return nil, FetchFunctionLogsResponse{}, fmt.Errorf("listing function logs: %w", err)
	}

	return nil, FetchFunctionLogsResponse{Logs: logs}, nil
}
