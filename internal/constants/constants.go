package constants

//nolint:gochecknoglobals
var (
	// Version is the current version of the server. It is set at build time.
	Version   = "dev"
	UserAgent = ProjectName + "/" + Version
)

const (
	ProjectName = "mcp-scaleway-functions"

	// TagCreatedByScalewayMCP is the tag added to all resources created by this MCP tool.
	// It's used as an extra safety measure to avoid deleting resources not created by this tool.
	TagCreatedByScalewayMCP = "created_by=" + ProjectName

	// RequiredPermissionSet is the minimum permission set required for the Scaleway API key.
	RequiredPermissionSet = "FunctionsFullAccess"
)
