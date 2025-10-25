package scaleway

import (
	"testing"

	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/fixed"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/mockscaleway"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTools_ListFunctions(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name           string
		givenFunctions []*function.Function
		givenError     error
		req            ListFunctionsRequest
		wantFunctions  []Function
		wantError      require.ErrorAssertionFunc
	}{
		{
			name: "success",
			givenFunctions: []*function.Function{
				{
					ID:          fixed.SomeFunctionID,
					Name:        fixed.SomeFunctionName,
					NamespaceID: fixed.SomeNamespaceID,
					Status:      function.FunctionStatusReady,
					Runtime:     function.FunctionRuntimePython313,
					DomainName:  "my-function-xyz.functions.fr-par.scw.cloud",
				},
			},
			wantFunctions: []Function{
				{
					ID:          fixed.SomeFunctionID,
					Status:      "ready",
					Name:        fixed.SomeFunctionName,
					NamespaceID: fixed.SomeNamespaceID,
					Runtime:     "python313",
					Endpoint:    "https://my-function-xyz.functions.fr-par.scw.cloud",
				},
			},
			wantError: require.NoError,
		},
		{
			name:       "api error",
			givenError: assert.AnError,
			wantError:  require.Error,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockFunctionsAPI := mockscaleway.NewMockFunctionAPI(t)

			mockFunctionsAPI.EXPECT().ListFunctions(mock.Anything, mock.Anything).
				Return(&function.ListFunctionsResponse{
					Functions: tc.givenFunctions,
				}, tc.givenError).Once()

			tools := &Tools{functionsAPI: mockFunctionsAPI}

			_, got, err := tools.ListFunctions(t.Context(), nil, tc.req)

			tc.wantError(t, err)

			if err != nil {
				return
			}

			assert.Equal(t, tc.wantFunctions, got.Functions)
		})
	}
}
