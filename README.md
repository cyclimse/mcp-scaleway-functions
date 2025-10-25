# MCP Scaleway Functions

Model Context Protocol (MCP) server to manage and deploy [Scaleway Serverless Functions](https://www.scaleway.com/en/serverless-functions/) using the [Model Context Protocol](https://modelcontextprotocol.org/) standard.

> [!CAUTION]
> This project is unofficial and not affiliated with or endorsed by Scaleway.
> Some small safety measures are in place to prevent the LLM from doing destructive actions,
> but they're not foolproof.
> Use at your own risk.

## Getting Started

Download the latest release from the [releases page](https://github.com/cyclimse/mcp-scaleway-functions/releases) or build it from source using Go.

Run the MCP server:

```console
./mcp-scaleway-functions
```

By default, the MCP server runs with the SSE transport on `http://localhost:8080`, but you can also change it to use Standard I/O (stdio) transport via the `--transport stdio` flag.

Then, configure your IDE or tool of choice to connect to the MCP server. Here are some examples:

### VSCode (sse example)

Add a new server configuration in your `.vscode/mcp.json` file:

```json
{
	"servers": {
		"mcp-scaleway-functions": {
			"url": "http://localhost:8080",
			"type": "http",
		}
	},
}
```

### Crush (stdio example)

Crush is an open-source coding agent that supports MCP. You can find more information about in the [Crush repository](https://github.com/charmbracelet/crush).

Add a new server configuration in your `~/.config/crush/crush.json` file:

```json
{
  "$schema": "https://charm.land/crush.json",
  "mcp": {
    "scaleway-functions": {
      "type": "stdio",
      "command": "mcp-scaleway-functions",
      "args": ["--transport", "stdio"],
      "timeout": 600,
      "disabled": false
    }
  }
}
```

You can even use Crush with [Scaleway Generative APIs](https://www.scaleway.com/en/generative-apis/) by adding a new provider in the same `~/.config/crush/crush.json` file:

```jsonc
{
  "mcp": {
	// ... see above ...
  },
  "providers": {
    "scaleway": {
      "name": "Scaleway",
      "base_url": "https://api.scaleway.ai/v1/",
      "type": "openai",
	  // To fetch from environment variables, use the `$VAR_NAME` syntax.
	  // Note: this key requires the "GenerativeApisModelAccess" permission.
      "api_key": "$SCW_SECRET_KEY",
      "models": [
        {
          "name": "Qwen coder",
          "id": "qwen3-coder-30b-a3b-instruct",
          "context_window": 128000,
          "default_max_tokens": 8000
        }
      ]
    }
  }
}
```

That's it 🎉! Have fun vibecoding and vibedevoopsing as you please.

## Configuration

By default, the MCP server reads from the standard Scaleway configuration file located at `~/.config/scw/config.yaml`.

Further configuration can be done via the
[Scaleway environment variables](https://www.scaleway.com/en/docs/scaleway-cli/reference-content/environment-variables/) to configure the MCP server.

For instance, you can set a region to work in via the `SCW_DEFAULT_REGION` environment variable.

```console
SCW_DEFAULT_REGION=nl-ams ./mcp-scaleway-functions
```

## Available Tools

| **Tool**                               | **Description**                                                                                                                   |
| -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `create_and_deploy_function_namespace` | Create and deploy a new function namespace.                                                                                       |
| `list_function_namespaces`             | List all function namespaces.                                                                                                     |
| `delete_function_namespace`            | Delete a function namespace.                                                                                                      |
| `list_functions`                       | List all functions in a namespace.                                                                                                |
| `list_function_runtimes`               | List all available function runtimes.                                                                                             |
| `create_and_deploy_function`           | Create and deploy a new function.                                                                                                 |
| `update_function`                      | Update the code or the configuration of an existing function.                                                                     |
| `delete_function`                      | Delete a function.                                                                                                                |
| `download_function`                    | Download the code of a function. This is useful to work on an existing function.                                                  |
| `fetch_function_logs`                  | Fetch the logs of a function.                                                                                                     |
| `add_dependency`                       | Add a dependency to a local function. Useful for dependencies that rely on native code and therefore need Docker to be installed. |

## Debugging

You can enable debug logging by using the `--debug` flag when starting the MCP server. This will log all requests and responses to/from the Scaleway API.

To configure the log level, use the `--log-level` flag (default is `info`). Available log levels are: `debug`, `info`, `warn`, `error`.

Logs are stored in the `$XDG_STATE_HOME/mcp-scaleway-functions` directory (usually `~/.local/state/mcp-scaleway-functions`).

## Development

Running tests:

```console
go tool gotestsum --format testdox
```

Generating mocks:

```console
go tool mockery
```

