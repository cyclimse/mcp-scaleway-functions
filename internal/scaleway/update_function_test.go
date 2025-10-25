package scaleway

import (
	"testing"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	"github.com/cyclimse/mcp-scaleway-functions/internal/testing/fixed"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/assert"
)

func TestUpdateFunctionRequest_ToSDK(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name          string
		in            UpdateFunctionRequest
		givenFunction *function.Function
		givenDigest   string
		wantSDKReq    *function.UpdateFunctionRequest
		wantError     assert.ErrorAssertionFunc
	}{
		{
			name: "some fields set",
			in: UpdateFunctionRequest{
				Description: scw.StringPtr("new description"),
				MinScale:    scw.Uint32Ptr(2),
			},
			givenFunction: &function.Function{
				ID:   "func-123",
				Name: "my-function",
				Tags: []string{
					"existing-tag",
					constants.TagCreatedByScalewayMCP,
				},
			},
			givenDigest: fixed.SomeCodeArchiveDigest,
			wantSDKReq: &function.UpdateFunctionRequest{
				FunctionID:  "func-123",
				Description: scw.StringPtr("new description"),
				MinScale:    scw.Uint32Ptr(2),
				Tags: scw.StringsPtr([]string{
					"existing-tag",
					constants.TagCreatedByScalewayMCP,
					constants.TagCodeArchiveDigestPrefix + fixed.SomeCodeArchiveDigest,
				}),
			},
			wantError: assert.NoError,
		},
		{
			name: "update existing code archive digest tag",
			in: UpdateFunctionRequest{
				Description: scw.StringPtr("another description"),
			},
			givenFunction: &function.Function{
				ID:   "func-123",
				Name: "my-function",
				Tags: []string{
					constants.TagCreatedByScalewayMCP,
					constants.TagCodeArchiveDigestPrefix + "old-digest",
				},
			},
			givenDigest: fixed.SomeCodeArchiveDigest,
			wantSDKReq: &function.UpdateFunctionRequest{
				FunctionID:  "func-123",
				Description: scw.StringPtr("another description"),
				Tags: scw.StringsPtr([]string{
					constants.TagCreatedByScalewayMCP,
					constants.TagCodeArchiveDigestPrefix + fixed.SomeCodeArchiveDigest,
				}),
			},
			wantError: assert.NoError,
		},
		{
			name: "do not set runtime if it's the same",
			in: UpdateFunctionRequest{
				Runtime: scw.StringPtr("python313"),
			},
			givenFunction: &function.Function{
				ID:      "func-123",
				Name:    "my-function",
				Runtime: function.FunctionRuntimePython313,
				Tags: []string{
					constants.TagCreatedByScalewayMCP,
				},
			},
			givenDigest: fixed.SomeCodeArchiveDigest,
			wantSDKReq: &function.UpdateFunctionRequest{
				FunctionID: "func-123",
				Tags: scw.StringsPtr([]string{
					constants.TagCreatedByScalewayMCP,
					constants.TagCodeArchiveDigestPrefix + fixed.SomeCodeArchiveDigest,
				}),
			},
			wantError: assert.NoError,
		},
		{
			name: "set runtime if it's changing",
			in: UpdateFunctionRequest{
				Runtime: scw.StringPtr("go124"),
			},
			givenFunction: &function.Function{
				ID:      "func-123",
				Name:    "my-function",
				Runtime: function.FunctionRuntimeGo122,
				Tags: []string{
					constants.TagCreatedByScalewayMCP,
				},
			},
			givenDigest: fixed.SomeCodeArchiveDigest,
			wantSDKReq: &function.UpdateFunctionRequest{
				FunctionID: "func-123",
				Runtime:    function.FunctionRuntimeGo124,
				Tags: scw.StringsPtr([]string{
					constants.TagCreatedByScalewayMCP,
					constants.TagCodeArchiveDigestPrefix + fixed.SomeCodeArchiveDigest,
				}),
			},
			wantError: assert.NoError,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.in.ToSDK(tc.givenFunction, tc.givenDigest)
			tc.wantError(t, err)

			if err != nil {
				return
			}

			assert.Equal(t, tc.wantSDKReq, got)
		})
	}
}
