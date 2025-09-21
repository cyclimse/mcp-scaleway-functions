package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/alecthomas/kong"
	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	"github.com/cyclimse/mcp-scaleway-functions/internal/scaleway"
	"github.com/lmittmann/tint"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	account "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// Version is the current version of the server. It is set at build time.
//
//nolint:gochecknoglobals
var Version = "dev"

type cliContext struct {
	Logger *slog.Logger
}

//nolint:gochecknoglobals
var cli struct {
	LogLevel slog.Level `help:"Log level (debug, info, warn, error)"`

	Serve serveCmd `cmd:"" default:"withargs" help:"Start the MCP server"`
}

type serveCmd struct {
	Profile string `help:"Scaleway profile to use (overrides the active profile)"`

	Transport string `default:"sse" enum:"sse,stdio" help:"Transport to use (sse or stdio)"`

	HTTPHost string `default:"localhost" help:"HTTP host to listen on"`
	HTTPPort int    `default:"8080"      help:"HTTP port to listen on"`
}

func (cmd *serveCmd) Run(cliCtx *cliContext) error {
	logger := cliCtx.Logger

	p, err := loadScalewayProfile(cmd.Profile)
	if err != nil {
		return fmt.Errorf("loading Scaleway profile: %w", err)
	}

	scwClient, err := scw.NewClient(scw.WithProfile(p))
	if err != nil {
		return fmt.Errorf("creating Scaleway client: %w", err)
	}

	if err := warnOnExcessivePermissions(context.Background(), logger, scwClient); err != nil {
		return fmt.Errorf("warning about permissions: %w", err)
	}

	tools := scaleway.NewTools(scwClient)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp_scaleway_functions",
		Title:   "MCP Scaleway Serverless Functions",
		Version: Version,
	}, nil)

	tools.Register(server)

	logger = logger.With(
		slog.String("transport", cmd.Transport),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	switch cmd.Transport {
	case "sse":
		return cmd.startSSE(ctx, logger, server)
	case "stdio":
		return cmd.startStdio(ctx, logger, server)
	default:
		//nolint:err113 // can't be caught anyway
		return fmt.Errorf("unknown transport: %s", cmd.Transport)
	}
}

//nolint:contextcheck // shutdown context does not inherit from parent, which is intentional
func (cmd *serveCmd) startSSE(ctx context.Context, logger *slog.Logger, server *mcp.Server) error {
	handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return server }, nil)
	addr := net.JoinHostPort(cmd.HTTPHost, strconv.Itoa(cmd.HTTPPort))

	httpServer := http.Server{
		Handler:           handler,
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("Starting server in SSE mode...", "addr", addr)

	go func() {
		<-ctx.Done()

		logger.Info("Shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		err := httpServer.Shutdown(
			shutdownCtx,
		)
		if err != nil {
			logger.Error("Error shutting down server", "error", err)
		}
	}()

	err := httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("starting HTTP server: %w", err)
	}

	return nil
}

func (*serveCmd) startStdio(
	ctx context.Context,
	logger *slog.Logger,
	server *mcp.Server,
) error {
	logger.Info("Starting server in stdio mode...")

	err := server.Run(ctx, &mcp.StdioTransport{})
	if err != nil {
		return fmt.Errorf("running server: %w", err)
	}

	return nil
}

func main() {
	w := os.Stderr

	ctx := kong.Parse(&cli)

	logger := slog.New(tint.NewHandler(w, nil))
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      cli.LogLevel,
			TimeFormat: time.Kitchen,
		}),
	))

	err := ctx.Run(&cliContext{Logger: logger})
	ctx.FatalIfErrorf(err)
}

func loadScalewayProfile(profileName string) (*scw.Profile, error) {
	cfg, err := scw.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	var p *scw.Profile

	if profileName != "" {
		// If the profile is overridden via the command line, we load it directly.
		p, err = cfg.GetProfile(profileName)
		if err != nil {
			return nil, fmt.Errorf("getting profile %q: %w", profileName, err)
		}
	} else {
		// Otherwise, we load the active profile.
		p, err = cfg.GetActiveProfile()
		if err != nil {
			return nil, fmt.Errorf("getting active profile: %w", err)
		}
	}

	// Finally, merge it with the environment variables overrides.
	p = scw.MergeProfiles(p, scw.LoadEnvProfile())

	return p, nil
}

func warnOnExcessivePermissions(
	ctx context.Context,
	logger *slog.Logger,
	scwClient *scw.Client,
) error {
	accountAPI := account.NewProjectAPI(scwClient)

	resp, err := accountAPI.ListProjects(&account.ProjectAPIListProjectsRequest{
		Page:     scw.Int32Ptr(1),
		PageSize: scw.Uint32Ptr(1),
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	// We don't have the permissions to list the projects, this is expected if the user
	// is using an API key with the minimal required permissions.
	if len(resp.Projects) == 0 {
		return nil
	}

	logger.WarnContext(
		ctx,
		"It seems that your Scaleway API key has permissions that are too open. "+
			`Consider creating a new API key with only the "`+constants.RequiredPermissionSet+`" permission set. `+
			"See: https://www.scaleway.com/en/docs/iam/reference-content/policy/ for more information.",
	)

	return nil
}
