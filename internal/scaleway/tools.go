package scaleway

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type Tools struct {
	scwClient    *scw.Client
	functionsAPI *function.API
}

func NewTools(scwClient *scw.Client) *Tools {
	return &Tools{
		scwClient:    scwClient,
		functionsAPI: function.NewAPI(scwClient),
	}
}

func (t *Tools) Register(s *mcp.Server) {
	// Namespace tools
	mcp.AddTool(s, createAndDeployFunctionNamespaceTool, t.CreateAndDeployFunctionNamespace)
	mcp.AddTool(s, listFunctionNamespacesTool, t.ListFunctionNamespaces)
	mcp.AddTool(s, deleteFunctionNamespaceTool, t.DeleteFunctionNamespace)

	// Function tools
	mcp.AddTool(s, listFunctionsTool, t.ListFunctions)
	mcp.AddTool(s, listFunctionRuntimesTool, t.ListFunctionRuntimes)

	mcp.AddTool(s, downloadFunctionTool, t.DownloadFunction)

	mcp.AddTool(s, createAndDeployFunctionTool, t.CreateAndDeployFunction)
	mcp.AddTool(s, updateFunctionTool, t.UpdateFunction)

	mcp.AddTool(s, deleteFunctionTool, t.DeleteFunction)
}
