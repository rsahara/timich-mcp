package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rsahara/timich-mcp/internal/app"
	"github.com/rsahara/timich-mcp/internal/state"
)

func TestToolErrorDoesNotIncludeStructuredContent(t *testing.T) {
	timich := app.New(state.NewStore(t.TempDir()), nil, t.TempDir())
	server := New(timich, "test")
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), serverTransport, nil); err != nil {
		t.Fatalf("server Connect() error = %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect() error = %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_search_capabilities",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool() protocol error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool() IsError = false, want true")
	}
	if result.StructuredContent != nil {
		t.Fatalf("StructuredContent = %#v, want nil", result.StructuredContent)
	}
	if len(result.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Content[0] = %T, want *mcp.TextContent", result.Content[0])
	}
	if !strings.Contains(text.Text, "not_paired") {
		t.Fatalf("error text = %q, want not_paired", text.Text)
	}
}
