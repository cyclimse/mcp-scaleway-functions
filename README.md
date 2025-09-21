# Unofficial MCP Server for Scaleway Functions

This is an unofficial implementation of a Model Context Protocol (MCP) server to manage and deploy [Scaleway Functions](https://www.scaleway.com/en/functions/) using the [Model Context Protocol (MCP)](https://modelcontextprotocol.org/) standard.

> [!CAUTION]
> This project is unofficial and not affiliated with or endorsed by Scaleway.
> Some small safety measures are in place to prevent the LLM from doing destructive actions,
> but they're not foolproof.
> Use at your own risk.

## Getting Started

Download the latest release from the [releases page](https://github.com/cyclimse/mcp_scaleway_functions/releases) or build it from source using Go.

Run the MCP server:

```bash
./mcp_scaleway_functions
```

By default, the MCP server runs with the SSE transport on `http://localhost:8080`, but you can also change it to use Standard I/O (stdio) transport via the `--transport stdio` flag.

Then, configure your IDE to use the MCP server. Here's an example with VSCode and GitHub Copilot:

```jsonc
// .vscode/mcp.json
{
	"servers": {
		"scaleway": {
			"url": "http://localhost:8080",
			"type": "http",
		}
	},
	"inputs": []
}
```

That's it 🎉! Have fun vibecoding and vibedevopsing as you please.

## Configuration

You can use the standard [Scaleway environment variables](https://www.scaleway.com/en/docs/scaleway-cli/reference-content/environment-variables/) to configure the MCP server.

For instance, you can set a region to work in via the `SCW_DEFAULT_REGION` environment variable.

```bash
SCW_DEFAULT_REGION=nl-ams ./mcp_scaleway_functions
```

## Available Tools

| **Tool**                               | **Description**                                                                  |
| -------------------------------------- | -------------------------------------------------------------------------------- |
| `create_and_deploy_function_namespace` | Create and deploy a new function namespace.                                      |
| `list_function_namespaces`             | List all function namespaces.                                                    |
| `delete_function_namespace`            | Delete a function namespace.                                                     |
| `list_functions`                       | List all functions in a namespace.                                               |
| `list_function_runtimes`               | List all available function runtimes.                                            |
| `create_and_deploy_function`           | Create and deploy a new function.                                                |
| `update_function`                      | Update the code or the configuration of an existing function.                    |
| `delete_function`                      | Delete a function.                                                               |
| `download_function`                    | Download the code of a function. This is useful to work on an existing function. |
