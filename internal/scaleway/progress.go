package scaleway

import (
	"context"
	"log/slog"
	"strings"

	"github.com/cyclimse/mcp-scaleway-functions/pkg/slogctx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
)

type FunctionDeploymentStep int

const (
	// From what I've seen, MCP progress can start at 0.
	StepCreatingCodeArchive FunctionDeploymentStep = iota
	StepUploadingCode
	StepBuildStarted
	StepBuildingFunction
	StepPushingFunctionImageToRegistry
	StepDeployingFunction

	TotalFunctionSteps
)

//nolint:gochecknoglobals
var emojiForStep = map[FunctionDeploymentStep]string{
	StepCreatingCodeArchive:            "📂",
	StepUploadingCode:                  "📤",
	StepBuildStarted:                   "🏗️",
	StepBuildingFunction:               "🛠️",
	StepPushingFunctionImageToRegistry: "📦",
	StepDeployingFunction:              "🚀",
}

type FunctionDeploymentProgress struct {
	functionName string
	currentStep  FunctionDeploymentStep
}

func NewFunctionDeploymentProgress(functionName string) *FunctionDeploymentProgress {
	return &FunctionDeploymentProgress{
		functionName: functionName,
		currentStep:  StepCreatingCodeArchive,
	}
}

func (p *FunctionDeploymentProgress) NotifyCodeArchiveCreation(
	ctx context.Context,
	req *mcp.CallToolRequest,
) {
	p.notifyInner(ctx, req, emojiForStep[StepCreatingCodeArchive]+" Creating code archive")
	p.incrementStep()
}

func (p *FunctionDeploymentProgress) NotifyCodeUploading(
	ctx context.Context,
	req *mcp.CallToolRequest,
) {
	p.notifyInner(ctx, req, emojiForStep[StepUploadingCode]+" Uploading code...")
	p.incrementStep()
}

func (p *FunctionDeploymentProgress) NotifyBuildStarted(
	ctx context.Context,
	req *mcp.CallToolRequest,
) {
	p.notifyInner(ctx, req, emojiForStep[StepBuildStarted]+" Starting build...")
	p.incrementStep()
}

func (p *FunctionDeploymentProgress) GetFunctionBuildCB(
	ctx context.Context,
	req *mcp.CallToolRequest,
) WaitForFunctionCallback {
	var lastBuildMessageNotified string
	// Reset to the step where the build starts.
	p.currentStep = StepBuildStarted

	return func(fun *function.Function) {
		buildMessage := valueOrDefault(fun.BuildMessage, "")
		hasChanged := buildMessage != lastBuildMessageNotified

		// If there is no build message, fallback to the function status.
		if buildMessage != "" && hasChanged {
			lastBuildMessageNotified = buildMessage

			p.notifyInner(ctx, req, displayBuildMessageWithEmoji(p.currentStep, buildMessage))
			p.incrementStep()
		}
	}
}

func displayBuildMessageWithEmoji(step FunctionDeploymentStep, message string) string {
	before, message, ok := strings.Cut(message, ":")
	if !ok {
		message = before
	}

	emoji, ok := emojiForStep[step]
	if !ok {
		return message
	}

	// Capitalize the first letter of the message.
	message = strings.TrimSpace(message)
	if message == "" {
		return emoji
	}

	message = strings.ToUpper(message[:1]) + message[1:]

	return emoji + " " + message
}

func (p *FunctionDeploymentProgress) incrementStep() {
	if p.currentStep < TotalFunctionSteps {
		p.currentStep++
	}
}

func (p *FunctionDeploymentProgress) notifyInner(
	ctx context.Context,
	req *mcp.CallToolRequest,
	message string,
) {
	logger := slogctx.FromContext(ctx).With(
		"function_name", p.functionName,
		"step", p.currentStep,
		"message", message,
	)
	progressToken := req.Params.GetProgressToken()

	params := &mcp.ProgressNotificationParams{
		Message:       message,
		ProgressToken: progressToken,
		Progress:      float64(p.currentStep),
		Total:         float64(TotalFunctionSteps),
	}

	logger.InfoContext(ctx, "Function deployment progressed")

	err := req.Session.NotifyProgress(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "Notifying progress", "error", err)
	}
}
