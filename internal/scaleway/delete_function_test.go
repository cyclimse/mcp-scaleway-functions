package scaleway

import (
	"testing"

	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/fixed"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/mockscaleway"
)

func TestTools_DeleteFunction(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name             string
		givenFunction    *function.Function
		givenDeleteError error
		req              DeleteFunctionRequest
		shouldDelete     bool
		wantError        require.ErrorAssertionFunc
	}{
		{
			name: "disallow deleting function not owned by tool",
			givenFunction: &function.Function{
				ID:   fixed.SomeFunctionID,
				Name: fixed.SomeFunctionName,
				// Missing tag that indicates ownership by the tool.
				Tags:   []string{"some-other-tag"},
				Status: function.FunctionStatusReady,
			},
			req: DeleteFunctionRequest{
				FunctionName: fixed.SomeFunctionName,
			},
			shouldDelete: false,
			wantError: func(t require.TestingT, err error, _ ...any) {
				assert.ErrorIs(t, err, ErrResourceNotOwnedByTool)
			},
		},
		{
			name: "success",
			givenFunction: &function.Function{
				ID:   fixed.SomeFunctionID,
				Name: fixed.SomeFunctionName,
				Tags: []string{constants.TagCreatedByScalewayMCP},
				// Function must be in a deletable state.
				Status: function.FunctionStatusReady,
			},
			req: DeleteFunctionRequest{
				FunctionName: fixed.SomeFunctionName,
			},
			shouldDelete: true,
			wantError:    require.NoError,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockFunctionsAPI := mockscaleway.NewMockFunctionAPI(t)

			mockFunctionsAPI.EXPECT().ListFunctions(&function.ListFunctionsRequest{
				Name: &tc.req.FunctionName,
			}, mock.Anything, mock.Anything).Return(&function.ListFunctionsResponse{
				Functions: []*function.Function{tc.givenFunction},
			}, nil).Once()

			if tc.shouldDelete {
				mockFunctionsAPI.EXPECT().DeleteFunction(&function.DeleteFunctionRequest{
					FunctionID: tc.givenFunction.ID,
				}, mock.Anything).Return(tc.givenFunction, tc.givenDeleteError)
			}

			tools := &Tools{functionsAPI: mockFunctionsAPI}

			_, _, err := tools.DeleteFunction(t.Context(), nil, tc.req)

			tc.wantError(t, err)

			if err != nil {
				return
			}
		})
	}
}
