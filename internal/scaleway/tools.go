package scaleway

import (
	"fmt"
	"sync"

	"github.com/cyclimse/mcp-scaleway-functions/internal/scaleway/cockpit"
	"github.com/moby/moby/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type Tools struct {
	scwClient    *scw.Client
	functionsAPI FunctionAPI

	cockpitClient cockpit.Client
	projectID     string

	// Docker client is only used for the "add_dependency" tool, and since initialization
	// can fail on some systems (e.g. when Docker is not installed/running), we only
	// initialize it when needed, and only once.
	loadDockerAPIOnce sync.Once
	dockerAPI         client.APIClient
}

//nolint:interfacebloat,inamedparam // Only meant for testing purposes.
type FunctionAPI interface {
	GetNamespace(
		*function.GetNamespaceRequest,
		...scw.RequestOption,
	) (*function.Namespace, error)
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

func NewTools(scwClient *scw.Client, projectID string) *Tools {
	return &Tools{
		scwClient:     scwClient,
		functionsAPI:  function.NewAPI(scwClient),
		cockpitClient: cockpit.NewClient(scwClient, projectID),
		projectID:     projectID,
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

	mcp.AddTool(s, fetchFunctionLogsTool, t.FetchFunctionLogs)

	// Dependency tools
	mcp.AddTool(s, addDependencyTool, t.AddDependency)
}

//nolint:nonamedreturns // actually like it this way.
func (t *Tools) loadDockerClient() (err error) {
	t.loadDockerAPIOnce.Do(func() {
		t.dockerAPI, err = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			err = fmt.Errorf("initializing docker client: %w", err)
		}
	})

	return err
}
