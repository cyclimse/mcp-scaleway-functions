package scaleway

import (
	"context"
	"fmt"
	"time"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

//nolint:gochecknoglobals
var createAndDeployFunctionTool = &mcp.Tool{
	Name: "create_and_deploy_function",
	Description: `Create and deploy a Scaleway Function from a local directory.
		
		- You **must** have already created a Namespace to deploy the function into and inject its ID via "namespace_name".
		- The directory **must** contain the function code.
		- The function runtime and handler **must** be specified in the request.

		Here's a Python example:
		
		"""python
		# In a file called handler.py
		def handle(event, context):
		  return {
		  	"body": {
		  		"message": 'Hello, world',
		  	},
		  	"statusCode": 200,
		  }
		"""

		The handler in this case would be "handler.handle" (file.function).`,
}

// We could embed function.CreateFunctionRequest but:
// - It seems the LLM is much better with `namespace_name` than `namespace_id`
// - The LLM seems to struggle with the `timeout` field which must be a string, but the fancy SDK type confuses it.
type CreateAndDeployFunctionRequest struct {
	Directory string `json:"directory"`

	// CreateFunctionRequest fields
	FunctionName               string            `json:"function_name"`
	NamespaceName              string            `json:"namespace_name"`
	Runtime                    string            `json:"runtime"`
	Handler                    string            `json:"handler"`
	Timeout                    string            `json:"timeout"`
	Description                string            `json:"description,omitempty"`
	Tags                       []string          `json:"tags,omitempty"`
	EnvironmentVariables       map[string]string `json:"environment_variables,omitempty"`
	SecretEnvironmentVariables map[string]string `json:"secret_environment_variables,omitempty"`
	MinScale                   *uint32           `json:"min_scale,omitempty"`
	MaxScale                   *uint32           `json:"max_scale,omitempty"`
	MemoryLimit                *uint32           `json:"memory_limit,omitempty"`
}

func (req CreateAndDeployFunctionRequest) ToSDK(
	namespaceID string,
) (*function.CreateFunctionRequest, error) {
	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil {
		return nil, fmt.Errorf("parsing timeout: %w", err)
	}

	secrets := make([]*function.Secret, 0, len(req.SecretEnvironmentVariables))
	for k, v := range req.SecretEnvironmentVariables {
		secrets = append(secrets, &function.Secret{
			Key:   k,
			Value: &v,
		})
	}

	return &function.CreateFunctionRequest{
		NamespaceID: namespaceID,
		Name:        req.FunctionName,
		Runtime:     function.FunctionRuntime(req.Runtime),
		Handler:     &req.Handler,
		Timeout: &scw.Duration{
			Seconds: int64(timeout.Seconds()),
		},
		Description:                &req.Description,
		Tags:                       setCreatedByTagIfAbsent(req.Tags),
		EnvironmentVariables:       &req.EnvironmentVariables,
		SecretEnvironmentVariables: secrets,
		MinScale:                   req.MinScale,
		MaxScale:                   req.MaxScale,
		MemoryLimit:                req.MemoryLimit,
	}, nil
}

func (t *Tools) CreateAndDeployFunction(
	ctx context.Context,
	req *mcp.CallToolRequest,
	in CreateAndDeployFunctionRequest,
) (*mcp.CallToolResult, Function, error) {
	progress := NewFunctionDeploymentProgress(in.FunctionName)

	ns, err := getFunctionNamespaceByName(ctx, t.functionsAPI, in.NamespaceName)
	if err != nil {
		return nil, Function{}, fmt.Errorf("getting namespace by name: %w", err)
	}

	createReq, err := in.ToSDK(ns.ID)
	if err != nil {
		return nil, Function{}, fmt.Errorf("converting to SDK request: %w", err)
	}

	// We always create the function first before zipping the code archive for
	// faster feedback to the user in case of errors.
	fun, err := t.functionsAPI.CreateFunction(createReq, scw.WithContext(ctx))
	if err != nil {
		return nil, Function{}, fmt.Errorf("creating function: %w", err)
	}

	progress.NotifyCodeArchiveCreation(ctx, req)

	archive, err := NewCodeArchive(in.Directory)
	if err != nil {
		return nil, Function{}, fmt.Errorf("creating archive: %w", err)
	}

	tags := append(fun.Tags, constants.TagCodeArchiveDigest+archive.Digest)

	// However, as a side-effect of doing creation first, we need to
	// update the function to add the code archive digest tag (which helps
	// avoid redeploying the same code in future updates).
	fun, err = t.functionsAPI.UpdateFunction(&function.UpdateFunctionRequest{
		FunctionID: fun.ID,
		Redeploy:   scw.BoolPtr(false),
		Tags:       scw.StringsPtr(tags),
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, Function{}, fmt.Errorf("updating function with code archive digest tag: %w", err)
	}

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

	_, err = t.functionsAPI.DeployFunction(&function.DeployFunctionRequest{
		FunctionID: fun.ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, Function{}, fmt.Errorf("deploying function: %w", err)
	}

	progress.NotifyBuildStarted(ctx, req)

	fun, err = waitForFunction(ctx, t.functionsAPI, fun.ID, progress.GetFunctionBuildCB(ctx, req))
	if err != nil {
		return nil, Function{}, fmt.Errorf("waiting for function to be ready: %w", err)
	}

	return nil, NewFunctionFromSDK(fun), nil
}
