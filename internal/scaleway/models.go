package scaleway

import (
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
)

type Namespace struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	ProjectID    string `json:"project_id"`
	Region       string `json:"region"`
}

func NewNamespaceFromSDK(n *function.Namespace) Namespace {
	return Namespace{
		ID:           n.ID,
		Name:         n.Name,
		Status:       n.Status.String(),
		ErrorMessage: valueOrDefault(n.ErrorMessage, ""),
		ProjectID:    n.ProjectID,
		Region:       n.Region.String(),
	}
}

type Runtime struct {
	function.Runtime
}

func NewRuntimeFromSDK(r *function.Runtime) Runtime {
	if r == nil {
		return Runtime{}
	}

	return Runtime{Runtime: *r}
}

type Function struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	Runtime      string `json:"runtime"`
	Endpoint     string `json:"endpoint,omitempty"`
}

func NewFunctionFromSDK(f *function.Function) Function {
	return Function{
		ID:           f.ID,
		Name:         f.Name,
		Description:  valueOrDefault(f.Description, ""),
		Status:       f.Status.String(),
		ErrorMessage: valueOrDefault(f.ErrorMessage, ""),
		Runtime:      f.Runtime.String(),
		Endpoint:     "https://" + f.DomainName,
	}
}

func valueOrDefault[T any](ptr *T, defaultValue T) T {
	if ptr != nil {
		return *ptr
	}

	return defaultValue
}
