package scaleway

import (
	"testing"

	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/fixed"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/mockscaleway"
)

func TestTools_CreateAndDeployFunctionNamespace(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name                  string
		givenCreatedNamespace *function.Namespace
		givenCreateError      error
		givenWaitError        error
		req                   CreateAndDeployFunctionNamespace
		wantNamespace         Namespace
		wantError             require.ErrorAssertionFunc
	}{
		{
			name: "success",
			givenCreatedNamespace: &function.Namespace{
				ID:        fixed.SomeNamespaceID,
				Name:      fixed.SomeNamespaceName,
				Status:    function.NamespaceStatusReady,
				ProjectID: fixed.SomeProjectID,
				Region:    fixed.SomeRegion,
			},
			req: CreateAndDeployFunctionNamespace{
				Name: fixed.SomeNamespaceName,
			},
			wantNamespace: Namespace{
				ID:        fixed.SomeNamespaceID,
				Name:      fixed.SomeNamespaceName,
				Status:    "ready",
				ProjectID: fixed.SomeProjectID,
				Region:    fixed.SomeRegion,
			},
			wantError: require.NoError,
		},
		{
			name:             "create error",
			givenCreateError: assert.AnError,
			wantError:        require.Error,
		},
		{
			name: "wait error",
			givenCreatedNamespace: &function.Namespace{
				ID: fixed.SomeNamespaceID,
			},
			givenWaitError: assert.AnError,
			req: CreateAndDeployFunctionNamespace{
				Name: fixed.SomeNamespaceName,
			},
			wantError: require.Error,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockFunctionsAPI := mockscaleway.NewMockFunctionAPI(t)

			mockFunctionsAPI.EXPECT().CreateNamespace(mock.Anything, mock.Anything).
				Return(tc.givenCreatedNamespace, tc.givenCreateError).Once()

			if tc.givenCreateError == nil {
				mockFunctionsAPI.EXPECT().WaitForNamespace(&function.WaitForNamespaceRequest{
					NamespaceID: tc.givenCreatedNamespace.ID,
				}, mock.Anything).Return(tc.givenCreatedNamespace, tc.givenWaitError)
			}

			tools := &Tools{functionsAPI: mockFunctionsAPI}

			_, gotNamespace, err := tools.CreateAndDeployFunctionNamespace(t.Context(), nil, tc.req)

			tc.wantError(t, err)

			if err != nil {
				return
			}

			assert.Equal(t, tc.wantNamespace, gotNamespace)
		})
	}
}
