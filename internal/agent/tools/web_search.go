package tools

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"charm.land/fantasy"
)

//go:embed web_search.md
var webSearchToolDescription []byte

// NewWebSearchTool creates a web search tool for sub-agents (no permissions needed).
func NewWebSearchTool(client *http.Client) fantasy.AgentTool {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}

	return fantasy.NewParallelAgentTool(
		WebSearchToolName,
		string(webSearchToolDescription),
		func(ctx context.Context, params WebSearchParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Query == "" {
				return fantasy.NewTextErrorResponse("query is required"), nil
			}

			maxResults := params.MaxResults
			if maxResults <= 0 {
				maxResults = 10
			}
			if maxResults > 20 {
				maxResults = 20
			}

			results, searchErr := searchDuckDuckGo(ctx, client, params.Query, maxResults)
			if searchErr != nil {
				slog.Debug("Web search failed", "error", searchErr, "query", params.Query, "max_results", maxResults)
				return fantasy.ToolResponse{}, fmt.Errorf("failed to search: %w", searchErr)
			}

			return fantasy.NewTextResponse(formatSearchResults(results)), nil
		})
}
