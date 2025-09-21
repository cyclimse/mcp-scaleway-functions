package constants

const (
	RequiredPermissionSet = "FunctionsFullAccess"

	// TagCreatedByScalewayMCP is the tag added to all resources created by this MCP tool.
	// It's used as an extra safety measure to avoid deleting resources not created by this tool.
	TagCreatedByScalewayMCP = "created_by=scaleway_mcp"
)
