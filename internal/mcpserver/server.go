package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rsahara/timich-mcp/internal/app"
)

func New(timich *app.App, version string) *mcp.Server {
	if version == "" {
		version = "dev"
	}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "timich-mcp",
		Title:   "Timich MCP",
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: "Search and preview photos/videos from a LAN Timich Agent paired with this local timich-mcp adapter.",
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_search_capabilities",
		Description: "Return the paired Timich Agent search capabilities. Pair first with the timich-mcp CLI if this tool reports not_paired.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, app.CapabilitiesOutput, error) {
		output, err := timich.GetSearchCapabilities(ctx)
		if err != nil {
			return nil, app.CapabilitiesOutput{}, err
		}
		return nil, output, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "search_assets",
		Description: "Search or browse Timich assets through the paired LAN Agent. " +
			"Use text for semantic/filename search, omit text for timeline browsing. " +
			"capturedAt.from/to must be UTC RFC3339 timestamps ending in Z. " +
			"page is zero-based and pageSize defaults to 20 with a maximum of 200.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input app.SearchAssetsInput) (*mcp.CallToolResult, app.SearchAssetsOutput, error) {
		output, err := timich.SearchAssets(ctx, input)
		if err != nil {
			return nil, app.SearchAssetsOutput{}, err
		}
		return nil, output, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_asset_preview",
		Description: "Fetch a preview image for a Timich asset id returned by search_assets and return a local file path.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input app.PreviewInput) (*mcp.CallToolResult, app.PreviewOutput, error) {
		output, err := timich.GetAssetPreview(ctx, input)
		if err != nil {
			return nil, app.PreviewOutput{}, err
		}
		return nil, output, nil
	})

	return server
}

func Run(ctx context.Context, timich *app.App, version string) error {
	return New(timich, version).Run(ctx, &mcp.StdioTransport{})
}
