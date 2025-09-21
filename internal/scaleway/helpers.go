package scaleway

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

var (
	ErrResourceNotFound       = errors.New("resource not found")
	ErrResourceNotOwnedByTool = errors.New("resource not owned by this tool")
)

func getFunctionNamespaceByName(
	ctx context.Context,
	functionAPI FunctionAPI,
	name string,
) (*function.Namespace, error) {
	resp, err := functionAPI.ListNamespaces(&function.ListNamespacesRequest{
		Name: &name,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}

	namespaces := resp.Namespaces

	if len(namespaces) == 0 {
		return nil, fmt.Errorf("%w: namespace %q", ErrResourceNotFound, name)
	}

	return namespaces[0], nil
}

func getFunctionByName(
	ctx context.Context,
	functionAPI FunctionAPI,
	name string,
) (*function.Function, error) {
	resp, err := functionAPI.ListFunctions(&function.ListFunctionsRequest{
		Name: &name,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("listing functions: %w", err)
	}

	functions := resp.Functions

	if len(functions) == 0 {
		return nil, fmt.Errorf("%w: function %q", ErrResourceNotFound, name)
	}

	return functions[0], nil
}

func setCreatedByTagIfAbsent(tags []string) []string {
	if !slices.Contains(tags, constants.TagCreatedByScalewayMCP) {
		tags = append(tags, constants.TagCreatedByScalewayMCP)
	}

	return tags
}

func checkResourceOwnership(tags []string) error {
	if !slices.Contains(tags, constants.TagCreatedByScalewayMCP) {
		return fmt.Errorf("%w: resource does not belong to this tool", ErrResourceNotOwnedByTool)
	}

	return nil
}

type WaitForFunctionCallback func(fun *function.Function)

// waitForFunction waits for a function to be in a terminal state (ready or error), running the
// provided callback on each polling iteration.
// Note: there is a nice [function.API.WaitForFunction] but it doesn't support a callback.
func waitForFunction(
	ctx context.Context,
	functionAPI FunctionAPI,
	functionID string,
	cb WaitForFunctionCallback,
) (*function.Function, error) {
	for {
		fun, err := functionAPI.GetFunction(&function.GetFunctionRequest{
			FunctionID: functionID,
		}, scw.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("getting function: %w", err)
		}

		if cb != nil {
			cb(fun)
		}

		terminalStatus := map[function.FunctionStatus]struct{}{
			function.FunctionStatusCreated: {},
			function.FunctionStatusError:   {},
			function.FunctionStatusLocked:  {},
			function.FunctionStatusReady:   {},
		}

		if _, isTerminal := terminalStatus[fun.Status]; isTerminal {
			return fun, nil
		}

		afterChan := time.After(2 * time.Second)

		// Sleep before polling again.
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf(
				"context done while waiting for function %q: %w",
				functionID,
				ctx.Err(),
			)
		case <-afterChan:
		}
	}
}
