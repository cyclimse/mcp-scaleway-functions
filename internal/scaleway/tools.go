package scaleway

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type Tools struct {
	scwClient    *scw.Client
	functionsAPI FunctionAPI
}

//nolint:interfacebloat,inamedparam // Only meant for testing purposes.
type FunctionAPI interface {
	CreateNamespace(
		*function.CreateNamespaceRequest,
		...scw.RequestOption,
	) (*function.Namespace, error)
	WaitForNamespace(
		*function.WaitForNamespaceRequest,
		...scw.RequestOption,
	) (*function.Namespace, error)
	ListNamespaces(
		*function.ListNamespacesRequest,
		...scw.RequestOption,
	) (*function.ListNamespacesResponse, error)
	DeleteNamespace(
		*function.DeleteNamespaceRequest,
		...scw.RequestOption,
	) (*function.Namespace, error)

	CreateFunction(
		*function.CreateFunctionRequest,
		...scw.RequestOption,
	) (*function.Function, error)
	DeployFunction(
		*function.DeployFunctionRequest,
		...scw.RequestOption,
	) (*function.Function, error)
	GetFunction(*function.GetFunctionRequest, ...scw.RequestOption) (*function.Function, error)
	ListFunctions(
		*function.ListFunctionsRequest,
		...scw.RequestOption,
	) (*function.ListFunctionsResponse, error)
	UpdateFunction(
		*function.UpdateFunctionRequest,
		...scw.RequestOption,
	) (*function.Function, error)
	DeleteFunction(
		*function.DeleteFunctionRequest,
		...scw.RequestOption,
	) (*function.Function, error)

	ListFunctionRuntimes(
		*function.ListFunctionRuntimesRequest,
		...scw.RequestOption,
	) (*function.ListFunctionRuntimesResponse, error)

	GetFunctionUploadURL(
		*function.GetFunctionUploadURLRequest,
		...scw.RequestOption,
	) (*function.UploadURL, error)
	GetFunctionDownloadURL(
		*function.GetFunctionDownloadURLRequest,
		...scw.RequestOption,
	) (*function.DownloadURL, error)
}

var _ FunctionAPI = (*function.API)(nil)

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

	// Dependency tools
	mcp.AddTool(s, addDependencyTool, t.AddDependency)
}
