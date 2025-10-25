package scaleway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cyclimse/mcp-scaleway-functions/pkg/slogctx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var updateFunctionTool = &mcp.Tool{
	Name: "update_function",
	Description: `Update the code or configuration of an existing Scaleway Function from a local directory.
		This can be useful to fix any mistakes you've made in the code.`,
}

// We could embed function.CreateFunctionRequest but:
// - It seems the LLM is much better with `function_name` than `function_id`.
type UpdateFunctionRequest struct {
	Directory    string `json:"directory"`
	FunctionName string `json:"function_name"`

	Runtime     *string   `json:"runtime,omitempty"`
	Handler     *string   `json:"handler,omitempty"`
	Timeout     *string   `json:"timeout,omitempty"`
	Description *string   `json:"description,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
	MinScale    *uint32   `json:"min_scale,omitempty"`
	MaxScale    *uint32   `json:"max_scale,omitempty"`
	MemoryLimit *uint32   `json:"memory_limit,omitempty"`
}

//nolint:funlen
func (req UpdateFunctionRequest) ToSDK(
	currentFunction *function.Function,
	codeArchiveDigest string,
) (*function.UpdateFunctionRequest, error) {
	var timeout *scw.Duration

	if req.Timeout != nil {
		d, err := time.ParseDuration(*req.Timeout)
		if err != nil {
			return nil, fmt.Errorf("parsing timeout: %w", err)
		}

		timeout = &scw.Duration{
			Seconds: int64(d.Seconds()),
		}
	}

	var tags *[]string

	if req.Tags != nil {
		toSet := make([]string, 0, len(*req.Tags)+1)

		toSet = append(toSet, *req.Tags...)
		toSet = setCreatedByTag(toSet)
		toSet = setCodeArchiveDigestTag(toSet, codeArchiveDigest)

		tags = &toSet
	} else {
		digest, _ := getCodeArchiveDigestFromTags(currentFunction.Tags)

		// We won't set req.Tags is if digest == currentDigest
		if digest != codeArchiveDigest {
			newTags := setCodeArchiveDigestTag(currentFunction.Tags, codeArchiveDigest)
			tags = &newTags
		}
	}

	var runtime function.FunctionRuntime

	if newRuntime := req.Runtime; newRuntime != nil {
		currentRuntime := currentFunction.Runtime.String()

		// Providing the API with the same runtime value
		// still results in a full redeploy, so only set it if it's changing.
		if !strings.EqualFold(*newRuntime, currentRuntime) {
			runtime = function.FunctionRuntime(*newRuntime)
		}
	}

	var handler *string

	if req.Handler != nil {
		// Providing the API with the same handler value
		// still results in a full redeploy, so only set it if it's changing.
		if *req.Handler != currentFunction.Handler {
			handler = req.Handler
		}
	}

	return &function.UpdateFunctionRequest{
		FunctionID:  currentFunction.ID,
		Runtime:     runtime,
		Handler:     handler,
		Timeout:     timeout,
		Description: req.Description,
		Tags:        tags,
		MinScale:    req.MinScale,
		MaxScale:    req.MaxScale,
		MemoryLimit: req.MemoryLimit,
	}, nil
}

//nolint:funlen
func (t *Tools) UpdateFunction(
	ctx context.Context,
	req *mcp.CallToolRequest,
	in UpdateFunctionRequest,
) (*mcp.CallToolResult, Function, error) {
	logger := slogctx.FromContext(ctx)
	progress := NewFunctionDeploymentProgress(in.FunctionName)

	fun, err := getFunctionByName(ctx, t.functionsAPI, in.FunctionName)
	if err != nil {
		return nil, Function{}, fmt.Errorf("getting function by name: %w", err)
	}

	if err := checkResourceOwnership(fun.Tags); err != nil {
		return nil, Function{}, err
	}

	progress.NotifyCodeArchiveCreation(ctx, req)

	archive, err := NewCodeArchive(in.Directory)
	if err != nil {
		return nil, Function{}, fmt.Errorf("creating archive: %w", err)
	}

	shouldUpload := true

	digest, found := getCodeArchiveDigestFromTags(fun.Tags)
	if found && archive.CompareDigest(digest) {
		logger.InfoContext(ctx, "code archive digest matches existing one, skipping upload",
			"function_name", in.FunctionName,
			"digest", digest,
		)

		shouldUpload = false
	}

	if shouldUpload {
		presignedURLResp, err := t.functionsAPI.GetFunctionUploadURL(
			&function.GetFunctionUploadURLRequest{
				FunctionID:    fun.ID,
				ContentLength: archive.Size,
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return nil, Function{}, fmt.Errorf("getting presigned URL: %w", err)
		}

		progress.NotifyCodeUploading(ctx, req)

		if err := archive.Upload(ctx, presignedURLResp.URL); err != nil {
			return nil, Function{}, fmt.Errorf("uploading archive: %w", err)
		}
	}

	updateReq, err := in.ToSDK(fun, archive.Digest)
	if err != nil {
		return nil, Function{}, fmt.Errorf("converting to SDK request: %w", err)
	}

	fun, err = t.functionsAPI.UpdateFunction(updateReq, scw.WithContext(ctx))
	if err != nil {
		return nil, Function{}, fmt.Errorf("updating function: %w", err)
	}

	if shouldUpload {
		progress.NotifyBuildStarted(ctx, req)
	}

	fun, err = waitForFunction(ctx, t.functionsAPI, fun.ID, progress.GetFunctionBuildCB(ctx, req))
	if err != nil {
		return nil, Function{}, fmt.Errorf("waiting for function to be ready: %w", err)
	}

	return nil, NewFunctionFromSDK(fun), nil
}
