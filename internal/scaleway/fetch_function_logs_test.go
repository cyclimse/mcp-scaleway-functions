package scaleway

import (
	"strings"
	"testing"

	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cyclimse/mcp-scaleway-functions/internal/scaleway/cockpit"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/fixed"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/mockcockpit"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/mockscaleway"
)

func TestTools_FetchFunctionLogs(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name            string
		givenFunction   *function.Function
		onNamespace     *function.Namespace
		req             FetchFunctionLogsRequest
		wantLogsFetched bool
		wantResp        FetchFunctionLogsResponse
		wantError       require.ErrorAssertionFunc
	}{
		{
			name:          "function not found",
			givenFunction: nil,
			req: FetchFunctionLogsRequest{
				FunctionName: fixed.SomeFunctionName,
			},
			wantError: func(t require.TestingT, err error, _ ...any) {
				assert.ErrorIs(t, err, ErrResourceNotFound)
			},
		},
		{
			// This is a current limitation: we cannot fetch logs if the function's
			// namespace does not belong to the same project as the active Scaleway profile.
			name: "namespace projectID does not match profile projectID",
			givenFunction: &function.Function{
				ID:          fixed.SomeFunctionID,
				Name:        fixed.SomeFunctionName,
				NamespaceID: fixed.SomeNamespaceID,
				DomainName:  "my-function-xyz.functions.fr-par.scw.cloud",
			},
			onNamespace: &function.Namespace{
				ID:        fixed.SomeNamespaceID,
				Name:      fixed.SomeNamespaceName,
				Status:    function.NamespaceStatusReady,
				ProjectID: "some-other-project-id",
				Region:    fixed.SomeRegion,
			},
			req: FetchFunctionLogsRequest{
				FunctionName: fixed.SomeFunctionName,
				StartTime:    fixed.SomeTimestampA,
				EndTime:      fixed.SomeTimestampB,
			},
			wantError: func(t require.TestingT, err error, _ ...any) {
				assert.ErrorContains(t, err, "does not support fetching logs in multiple Scaleway projects")
			},
		},
		{
			name: "success",
			givenFunction: &function.Function{
				ID:          fixed.SomeFunctionID,
				Name:        fixed.SomeFunctionName,
				NamespaceID: fixed.SomeNamespaceID,
			},
			onNamespace: &function.Namespace{
				ID:        fixed.SomeNamespaceID,
				Name:      fixed.SomeNamespaceName,
				Status:    function.NamespaceStatusReady,
				ProjectID: fixed.SomeProjectID,
				Region:    fixed.SomeRegion,
			},
			req: FetchFunctionLogsRequest{
				FunctionName: fixed.SomeFunctionName,
				StartTime:    fixed.SomeTimestampA,
				EndTime:      fixed.SomeTimestampB,
			},
			wantLogsFetched: true,
			wantResp: FetchFunctionLogsResponse{
				Logs: []cockpit.Log{
					{Timestamp: fixed.SomeTimestampA, Message: "Function started"},
					{Timestamp: fixed.SomeTimestampB, Message: "Handling request"},
				},
			},
			wantError: require.NoError,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockFunctionsAPI := mockscaleway.NewMockFunctionAPI(t)
			mockCockpitClient := mockcockpit.NewMockClient(t)

			if tc.givenFunction != nil {
				mockFunctionsAPI.EXPECT().ListFunctions(mock.Anything, mock.Anything).Return(
					&function.ListFunctionsResponse{
						Functions: []*function.Function{tc.givenFunction},
					},
					nil,
				).Once()

				mockFunctionsAPI.EXPECT().
					GetNamespace(&function.GetNamespaceRequest{
						NamespaceID: tc.givenFunction.NamespaceID,
					}, mock.Anything).
					Return(tc.onNamespace, nil).
					Once()
			} else {
				mockFunctionsAPI.EXPECT().ListFunctions(mock.Anything, mock.Anything).Return(
					&function.ListFunctionsResponse{
						Functions: []*function.Function{},
					},
					nil,
				).Once()
			}

			if tc.wantLogsFetched {
				resourceName, _, _ := strings.Cut(tc.givenFunction.DomainName, ".")

				mockCockpitClient.EXPECT().
					ListFunctionLogs(
						mock.Anything,
						resourceName,
						tc.req.StartTime,
						tc.req.EndTime,
					).
					Return(tc.wantResp.Logs, nil).
					Once()
			}

			tools := &Tools{
				functionsAPI:  mockFunctionsAPI,
				cockpitClient: mockCockpitClient,
				projectID:     fixed.SomeProjectID,
			}

			_, resp, err := tools.FetchFunctionLogs(t.Context(), nil, tc.req)

			tc.wantError(t, err)
			if err != nil {
				return
			}

			assert.Equal(t, tc.wantResp, resp)
		})
	}
}
