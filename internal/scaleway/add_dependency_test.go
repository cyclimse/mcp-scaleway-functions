package scaleway

import (
	"io"
	"os"
	"sync"
	"testing"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/fixed"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/mockdocker"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/mockscaleway"
	"github.com/moby/moby/api/types/container"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDockerImageReader struct{}

func (*mockDockerImageReader) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (*mockDockerImageReader) Close() error {
	return nil
}

var listRuntimesResponse = &function.ListFunctionRuntimesResponse{
	// Reference: scw function runtime list
	Runtimes: []*function.Runtime{
		{
			Name:     "python3.13",
			Language: "Python",
			Version:  "3.13",
		},
		{
			Name:     "node22",
			Language: "Node",
			Version:  "22",
		},
		{
			// We support rust as a runtime, but not for adding dependencies
			// through this tool.
			Name:     "rust1.85",
			Language: "Rust",
		},
	},
}

func TestTools_AddDependency(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()

	myNodeFunctionDir := rootDir + "/my-node-function"
	myPythonFunctionDir := rootDir + "/my-python-function"
	myRustFunctionDir := rootDir + "/my-rust-function"

	for _, dir := range []string{
		myNodeFunctionDir,
		myPythonFunctionDir,
		myRustFunctionDir,
	} {
		err := os.Mkdir(dir, 0o750)
		require.NoError(t, err)
	}

	tt := []struct {
		name            string
		req             AddDependencyRequest
		wantPulledImage string
		wantCmd         []string
		wantError       require.ErrorAssertionFunc
	}{
		{
			name: "success with node",
			req: AddDependencyRequest{
				Directory: myNodeFunctionDir,
				Runtime:   "node22",
				Package:   "sharp",
			},
			wantPulledImage: "node:22-alpine",
			wantCmd: []string{
				"npm",
				"install",
				"sharp",
				"--prefix",
				"/function",
			},
			wantError: require.NoError,
		},
		{
			name: "success with python",
			req: AddDependencyRequest{
				Directory: myPythonFunctionDir,
				Runtime:   "python3.13",
				Package:   "requests",
			},
			wantPulledImage: constants.PublicRuntimesRegistry + "/python-dep:3.13",
			wantCmd: []string{
				"pip",
				"install",
				"requests",
				"--target",
				"/function/" + constants.PythonPackageFolder,
			},
			wantError: require.NoError,
		},
		{
			name: "unsupported runtime",
			req: AddDependencyRequest{
				Directory: myRustFunctionDir,
				Runtime:   "rust1.85",
				Package:   "serde",
			},
			wantError: func(tt require.TestingT, err error, _ ...any) {
				require.ErrorIs(tt, err, ErrRuntimeDependencyNotSupported)
			},
		},
		{
			name: "runtime not found",
			req: AddDependencyRequest{
				Directory: myPythonFunctionDir,
				Runtime:   "nonexistent-runtime",
				Package:   "requests",
			},
			wantError: func(tt require.TestingT, err error, _ ...any) {
				require.ErrorIs(tt, err, ErrRuntimeNotFound)
			},
		},
		{
			name: "directory does not exist",
			req: AddDependencyRequest{
				Directory: "invalid-directory",
				Runtime:   "python3.13",
				Package:   "requests",
			},
			wantError: func(tt require.TestingT, err error, _ ...any) {
				require.ErrorIs(tt, err, ErrDirectoryNotExist)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockFunctionsAPI := mockscaleway.NewMockFunctionAPI(t)
			mockDockerAPI := mockdocker.NewMockAPIClient(t)

			tools := &Tools{
				functionsAPI:      mockFunctionsAPI,
				dockerAPI:         mockDockerAPI,
				loadDockerAPIOnce: sync.Once{},
			}
			// A bit ugly: we force the loadDockerAPIOnce to be "done" so that
			// it returns the existing mockDockerAPI instead of trying to initialize
			// a real Docker client.
			tools.loadDockerAPIOnce.Do(func() {})

			mockFunctionsAPI.EXPECT().
				ListFunctionRuntimes(&function.ListFunctionRuntimesRequest{}, mock.Anything).
				Return(listRuntimesResponse, nil).
				Once()

			if tc.wantPulledImage != "" {
				mockDockerAPI.EXPECT().ImagePull(mock.Anything, tc.wantPulledImage, mock.Anything).
					Return(&mockDockerImageReader{}, nil).Once()
			}

			if tc.wantCmd != nil {
				mockDockerAPI.EXPECT().ContainerCreate(mock.Anything, mock.MatchedBy(
					func(config *container.Config) bool {
						return assert.Equal(t, tc.wantCmd, config.Cmd)
					}),
					mock.Anything, mock.Anything, mock.Anything, mock.Anything,
				).Return(container.CreateResponse{
					ID: fixed.SomeDockerContainerID,
				}, nil).Once()

				mockDockerAPI.EXPECT().
					ContainerStart(mock.Anything, fixed.SomeDockerContainerID, mock.Anything).
					Return(nil).
					Once()

				waitRespChan := make(chan container.WaitResponse)
				errChan := make(chan error)

				mockDockerAPI.EXPECT().
					ContainerWait(mock.Anything, fixed.SomeDockerContainerID, mock.Anything).
					Return(
						waitRespChan,
						errChan,
					).
					Once()

				// Simulate the container finishing successfully
				go func() {
					waitRespChan <- container.WaitResponse{
						StatusCode: 0,
					}

					close(waitRespChan)
					close(errChan)
				}()
			}

			_, _, err := tools.AddDependency(t.Context(), nil, tc.req)

			tc.wantError(t, err)
		})
	}
}
