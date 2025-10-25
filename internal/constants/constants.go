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

	// TagCodeArchiveDigestPrefix is the tag key used to store the code archive digest.
	// It's used to avoid redeploying the same code.
	TagCodeArchiveDigestPrefix = "code_archive_digest="

	// RequiredPermissionSets is the minimum permission sets required for the Scaleway API key.
	RequiredPermissionSets = "FunctionsFullAccess, ObservabilityFullAccess"

	PublicRuntimesRegistry = "rg.fr-par.scw.cloud/scwfunctionsruntimes-public"
	PythonPackageFolder    = "package"
)
