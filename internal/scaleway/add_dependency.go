package scaleway

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

var (
	ErrRuntimeNotFound               = fmt.Errorf("runtime not found")
	ErrRuntimeDependencyNotSupported = fmt.Errorf("adding dependencies is not supported for this runtime")

	supportedLanguagesForDependencies = []string{
		"python",
		"node",
	}

	loadDockerClientOnce sync.Once
	dockerClient         *client.Client
)

//nolint:gochecknoglobals
var addDependencyTool = &mcp.Tool{
	Name: "add_dependency",
	Description: `Add a dependency to a Scaleway Function.
	This uses the recommended way to include dependencies in a Scaleway Function, depending on the runtime.
	The "package" argument is the name of the package to add, for example "requests" for Python or "axios" for Node.js.
	This is essential for dependencies that rely on native libraries like "numpy" for Python or "sharp" for Node.js.
	The provided "directory" must be an existing directory where the function code is located.`,
}

type AddDependencyRequest struct {
	Directory string `json:"directory"`
	Runtime   string `json:"runtime"`
	Package   string `json:"package"`
}

type AddDependencyResponse struct{}

func (t *Tools) AddDependency(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in AddDependencyRequest,
) (*mcp.CallToolResult, AddDependencyResponse, error) {
	// Check that the runtime exists and supports dependencies
	runtimes, err := t.functionsAPI.ListFunctionRuntimes(&function.ListFunctionRuntimesRequest{}, scw.WithContext(ctx))
	if err != nil {
		return nil, AddDependencyResponse{}, fmt.Errorf("listing function runtimes: %w", err)
	}

	i := slices.IndexFunc(runtimes.Runtimes, func(r *function.Runtime) bool {
		return r.Name == in.Runtime
	})
	if i == -1 {
		return nil, AddDependencyResponse{}, fmt.Errorf("%w: %s", ErrRuntimeNotFound, in.Runtime)
	}

	runtime := runtimes.Runtimes[i]
	language := strings.ToLower(runtime.Language)

	if !slices.Contains(supportedLanguagesForDependencies, language) {
		return nil, AddDependencyResponse{}, fmt.Errorf("%w: %s", ErrRuntimeDependencyNotSupported, in.Runtime)
	}

	// Check that the directory exists
	if _, err := os.Stat(in.Directory); os.IsNotExist(err) {
		return nil, AddDependencyResponse{}, fmt.Errorf("directory does not exist: %s", in.Directory)
	}

	dockerClient, err := loadDockerClient()
	if err != nil {
		return nil, AddDependencyResponse{}, fmt.Errorf("loading docker client: %w", err)
	}

	var containerConfig *container.Config
	var hostConfig *container.HostConfig

	switch language {
	case "python":
		// Create package folder if it doesn't exist to avoid permissions issues
		if err := os.MkdirAll(in.Directory+"/"+constants.PythonPackageFolder, 0o755); err != nil {
			return nil, AddDependencyResponse{}, fmt.Errorf("creating package folder: %w", err)
		}

		containerConfig, hostConfig = getPythonContainerConfigs(runtime, in.Directory, in.Package)
	// case "node":
	// 	containerConfig, hostConfig, err = getNodeContainerConfigs(in.Directory, in.Package)
	default:
		return nil, AddDependencyResponse{}, fmt.Errorf("%w: %s", ErrRuntimeDependencyNotSupported, in.Runtime)
	}

	if err := runContainer(ctx, dockerClient, containerConfig, hostConfig); err != nil {
		return nil, AddDependencyResponse{}, fmt.Errorf("running container: %w", err)
	}

	return &mcp.CallToolResult{}, AddDependencyResponse{}, nil
}

func loadDockerClient() (*client.Client, error) {
	var err error

	loadDockerClientOnce.Do(func() {
		dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})
	if err != nil {
		return nil, err
	}

	return dockerClient, nil
}

// Reference: https://www.scaleway.com/en/docs/serverless-functions/how-to/package-function-dependencies-in-zip/?tab=python-2
func getPythonContainerConfigs(runtime *function.Runtime, directory, pkg string) (*container.Config, *container.HostConfig) {
	return &container.Config{
			Image: constants.PublicRuntimesRegistry + "/python-dep:" + runtime.Version,
			Cmd:   []string{"pip", "install", pkg, "-t", "/function/" + constants.PythonPackageFolder},
			Env: []string{
				"PYTHONUNBUFFERED=1",
			},
			WorkingDir: "/function",
		}, &container.HostConfig{
			Binds: []string{
				directory + "/:/function:rw",
			},
			AutoRemove: true,
		}
}

func runContainer(
	ctx context.Context,
	dockerClient *client.Client,
	containerConfig *container.Config,
	hostConfig *container.HostConfig,
) error {
	// TODO: this is really slow: we should send progress updates to the user
	// when pulling the image and running the container
	reader, err := dockerClient.ImagePull(ctx, containerConfig.Image, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %s: %w", containerConfig.Image, err)
	}
	defer reader.Close()

	// We don't need to read the output, just wait for the image to be pulled.
	// TODO: add a timeout?
	_, _ = io.Copy(io.Discard, reader)

	resp, err := dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("creating container: %w", err)
	}

	if err := dockerClient.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}

	statusCh, errCh := dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("waiting for container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("container exited with status code %d", status.StatusCode)
		}
	}

	return nil
}
