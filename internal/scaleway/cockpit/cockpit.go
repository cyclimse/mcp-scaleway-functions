package cockpit

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cyclimse/mcp-scaleway-functions/internal/constants"
	cockpit "github.com/scaleway/scaleway-sdk-go/api/cockpit/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	tokenName = constants.ProjectName

	// queryTemplateServerless is the template used to query logs from Loki for Serverless Functions & Containers.
	// Because Serverless logs are sent as JSON, we only get the message field.
	queryTemplateServerless = `{resource_name="%s", resource_type="%s"} |~ "^{.*}$" | json | line_format "{{.message}}"`
)

var (
	ErrNoScalewayLogsDataSource = errors.New(
		"no Scaleway logs data source found; please wait a few minutes and try again",
	)
	ErrTokenHasNoSecretKey = errors.New("token has no secret key")
)

type Log struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type Client interface {
	ListFunctionLogs(
		ctx context.Context,
		resourceName string,
		start time.Time,
		end time.Time,
	) ([]Log, error)
	// note(cyclimse): makes me think we should have a buildID in Scaleway Functions build logs
	//                 to link logs to a specific build.
	ListFunctionBuildLogs(
		ctx context.Context,
		resourceName string,
		start time.Time,
		end time.Time,
	) ([]Log, error)
}

type client struct {
	cockpitAPI *cockpit.RegionalAPI
	projectID  string

	initLokiClientOnce sync.Once
	lokiClient         LokiClient
}

func NewClient(scwClient *scw.Client, projectID string) Client {
	return &client{
		cockpitAPI: cockpit.NewRegionalAPI(scwClient),
		projectID:  projectID,
	}
}

// ListFunctionLogs implements Client.
func (c *client) ListFunctionLogs(
	ctx context.Context,
	resourceName string,
	start time.Time,
	end time.Time,
) ([]Log, error) {
	lokiClient, err := c.getLokiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting Loki client: %w", err)
	}

	logs, err := lokiClient.Query(
		ctx,
		fmt.Sprintf(queryTemplateServerless, resourceName, "serverless_function"),
		start,
		end,
	)
	if err != nil {
		return nil, fmt.Errorf("querying logs: %w", err)
	}

	return logs, nil
}

// ListFunctionBuildLogs implements Client.
func (*client) ListFunctionBuildLogs(
	_ context.Context,
	_ string,
	_ time.Time,
	_ time.Time,
) ([]Log, error) {
	panic("unimplemented")
}

//nolint:nonamedreturns // actually like it this way.
func (c *client) getLokiClient(ctx context.Context) (lokiClient LokiClient, err error) {
	c.initLokiClientOnce.Do(func() {
		var (
			dataSource string
			token      string
		)

		dataSource, err = c.getScalewayLogsDataSourceURL(ctx)
		if err != nil {
			err = fmt.Errorf(
				"getting Scaleway logs data source for project %q: %w",
				c.projectID,
				err,
			)

			return
		}

		token, err = c.createToken(ctx)
		if err != nil {
			err = fmt.Errorf("creating token: %w", err)

			return
		}

		c.lokiClient = NewLokiClient(dataSource, token)
	})

	return c.lokiClient, err
}

func (c *client) createToken(ctx context.Context) (string, error) {
	resp, err := c.cockpitAPI.ListTokens(&cockpit.RegionalAPIListTokensRequest{
		TokenScopes: []cockpit.TokenScope{cockpit.TokenScopeReadOnlyLogs},
		ProjectID:   c.projectID,
	}, scw.WithAllPages(), scw.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("listing tokens: %w", err)
	}

	i := slices.IndexFunc(resp.Tokens, func(t *cockpit.Token) bool {
		return t.Name == tokenName
	})

	if i != -1 {
		// Unfortunately, the SecretKey is only shown once. So we're going to have to
		// delete and recreate it.
		err := c.cockpitAPI.DeleteToken(&cockpit.RegionalAPIDeleteTokenRequest{
			TokenID: resp.Tokens[i].ID,
		}, scw.WithContext(ctx))
		if err != nil {
			return "", fmt.Errorf("deleting existing token: %w", err)
		}
	}

	token, err := c.cockpitAPI.CreateToken(&cockpit.RegionalAPICreateTokenRequest{
		Name:        tokenName,
		TokenScopes: []cockpit.TokenScope{cockpit.TokenScopeReadOnlyLogs},
	}, scw.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("creating token: %w", err)
	}

	if token.SecretKey == nil {
		return "", ErrTokenHasNoSecretKey
	}

	return *token.SecretKey, nil
}

func (c *client) getScalewayLogsDataSourceURL(ctx context.Context) (string, error) {
	resp, err := c.cockpitAPI.ListDataSources(&cockpit.RegionalAPIListDataSourcesRequest{
		Origin:    cockpit.DataSourceOriginScaleway,
		Types:     []cockpit.DataSourceType{cockpit.DataSourceTypeLogs},
		ProjectID: c.projectID,
		// There should be at most one such data source.
		Page:     scw.Int32Ptr(1),
		PageSize: scw.Uint32Ptr(1),
	}, scw.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("listing data sources: %w", err)
	}

	if len(resp.DataSources) == 0 {
		return "", ErrNoScalewayLogsDataSource
	}

	dataSource := resp.DataSources[0]

	return dataSource.URL, nil
}
