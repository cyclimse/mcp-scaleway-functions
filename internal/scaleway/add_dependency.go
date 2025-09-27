package scaleway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	"github.com/cyclimse/mcp-scaleway-functions/internal/std"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

var (
	ErrRuntimeNotFound               = errors.New("runtime not found")
	ErrRuntimeDependencyNotSupported = errors.New(
		"adding dependencies is not supported for this runtime",
	)
	ErrDirectoryNotExist       = errors.New("directory does not exist")
	ErrDependencyInstallFailed = errors.New(
		"failed to install dependency",
	)

	//nolint:gochecknoglobals
	supportedLanguagesForDependencies = []string{
		"python",
		"node",
	}
)

//nolint:gochecknoglobals
var addDependencyTool = &mcp.Tool{
	Name: "add_dependency",
	Description: `Add a native dependency to a Scaleway Function.
	This uses a alpine-based Docker container to install the dependency in the function directory.
	The "package" argument is the name of the package to add, for example "pydantic" for Python or "sharp" for Node.js.
	
	Note that for non-native dependencies you can (and may favor) add them through:
	  - Python: "pip install <package> --target ./<function_directory>/package"
	  - Node.js: "npm install <package> --prefix ./<function_directory>"

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
	runtime, language, err := getAndValidateRuntime(ctx, t.functionsAPI, in)
	if err != nil {
		return nil, AddDependencyResponse{}, fmt.Errorf("getting runtime: %w", err)
	}

	// Check that the directory exists
	if _, err := os.Stat(in.Directory); os.IsNotExist(err) {
		return nil, AddDependencyResponse{}, fmt.Errorf(
			"%w: %s",
			ErrDirectoryNotExist,
			in.Directory,
		)
	}

	// Initialize Docker client if needed
	err = t.loadDockerClient()
	if err != nil {
		return nil, AddDependencyResponse{}, fmt.Errorf("loading docker client: %w", err)
	}

	var (
		containerConfig *container.Config
		hostConfig      *container.HostConfig
	)

	switch language {
	case "python":
		// Create package folder if it doesn't exist to avoid permissions issues
		if err := os.MkdirAll(in.Directory+"/"+constants.PythonPackageFolder, 0o750); err != nil {
			return nil, AddDependencyResponse{}, fmt.Errorf("creating package folder: %w", err)
		}

		containerConfig, hostConfig = getPythonContainerConfigs(runtime, in.Directory, in.Package)
	case "node":
		containerConfig, hostConfig = getNodeContainerConfigs(
			runtime,
			in.Directory,
			in.Package,
		)
	default:
		return nil, AddDependencyResponse{}, fmt.Errorf(
			"%w: %s",
			ErrRuntimeDependencyNotSupported,
			in.Runtime,
		)
	}

	if err := runContainer(ctx, t.dockerAPI, containerConfig, hostConfig); err != nil {
		return nil, AddDependencyResponse{}, fmt.Errorf("running container: %w", err)
	}

	return &mcp.CallToolResult{}, AddDependencyResponse{}, nil
}

func getAndValidateRuntime(
	ctx context.Context,
	functionsAPI FunctionAPI,
	in AddDependencyRequest,
) (*function.Runtime, string, error) {
	runtimes, err := functionsAPI.ListFunctionRuntimes(
		&function.ListFunctionRuntimesRequest{},
		scw.WithContext(ctx),
	)
	if err != nil {
		return nil, "", fmt.Errorf("listing function runtimes: %w", err)
	}

	i := slices.IndexFunc(runtimes.Runtimes, func(r *function.Runtime) bool {
		return r.Name == in.Runtime
	})
	if i == -1 {
		return nil, "", fmt.Errorf("%w: %s", ErrRuntimeNotFound, in.Runtime)
	}

	runtime := runtimes.Runtimes[i]
	language := strings.ToLower(runtime.Language)

	if !slices.Contains(supportedLanguagesForDependencies, language) {
		return nil, "", fmt.Errorf("%w: %s", ErrRuntimeDependencyNotSupported, in.Runtime)
	}

	return runtime, language, nil
}

// Reference: https://www.scaleway.com/en/docs/serverless-functions/how-to/package-function-dependencies-in-zip/?tab=python-2
func getPythonContainerConfigs(
	runtime *function.Runtime,
	directory, pkg string,
) (*container.Config, *container.HostConfig) {
	return &container.Config{
			Image: constants.PublicRuntimesRegistry + "/python-dep:" + runtime.Version,
			Cmd: []string{
				"pip",
				"install",
				pkg,
				"--target",
				"/function/" + constants.PythonPackageFolder,
			},
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

func getNodeContainerConfigs(
	runtime *function.Runtime,
	directory, pkg string,
) (*container.Config, *container.HostConfig) {
	// Stangely enough, we don't provide a Scaleway-specific image for Node.js dependencies
	// like we do for Python. So we just use the public Node.js Alpine-based image from Docker Hub.
	versionParts := strings.SplitN(runtime.Version, ".", 2)
	if len(versionParts) == 0 {
		versionParts = []string{runtime.Version}
	}

	majorVersion := versionParts[0]
	image := "node:" + majorVersion + "-alpine"

	return &container.Config{
			Image: image,
			Cmd: []string{
				"npm",
				"install",
				pkg,
				"--prefix",
				"/function",
			},
			Env: []string{
				// Do not install dev dependencies!
				"NODE_ENV=production",
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
	dockerClient client.APIClient,
	containerConfig *container.Config,
	hostConfig *container.HostConfig,
) error {
	// LATER: this is really slow: we should send progress updates to the user
	// when pulling the image and running the container
	reader, err := dockerClient.ImagePull(ctx, containerConfig.Image, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %s: %w", containerConfig.Image, err)
	}

	defer func() {
		_ = reader.Close()
	}()

	// We don't need to read the output, just wait for the image to be pulled.
	err = std.Copy(ctx, io.Discard, reader)
	if err != nil {
		return fmt.Errorf("reading image pull response: %w", err)
	}

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
			return fmt.Errorf("%w: exit code %d", ErrDependencyInstallFailed, status.StatusCode)
		}
	}

	return nil
}
