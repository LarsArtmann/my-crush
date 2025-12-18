package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/filepathext"
	"github.com/charmbracelet/crush/internal/permission"
)

//go:embed memu.md
var memuDescription []byte

// MemUMemoryService represents a simplified interface for memory operations
type MemUMemoryService interface {
	Memorize(ctx context.Context, resourceURL, modality string) error
	Retrieve(ctx context.Context, queries []string) ([]string, error)
	Search(ctx context.Context, query string) ([]string, error)
	Forget(ctx context.Context, query string) error
}

// Mock memory service for development/testing
type MockMemUService struct{}

func (m *MockMemUService) Memorize(ctx context.Context, resourceURL, modality string) error {
	// In a real implementation, this would process resource and extract memories
	// For now, just log the operation to provide feedback
	return nil
}

func (m *MockMemUService) Retrieve(ctx context.Context, queries []string) ([]string, error) {
	// Mock retrieval - in real implementation this would use MemU's actual retrieval
	memories := []string{
		fmt.Sprintf("MemU would retrieve relevant memories for: %s", strings.Join(queries, ", ")),
	}
	return memories, nil
}

func (m *MockMemUService) Search(ctx context.Context, query string) ([]string, error) {
	// Mock search - in real implementation this would use MemU's actual search
	results := []string{
		fmt.Sprintf("MemU would search for: %s", query),
	}
	return results, nil
}

func (m *MockMemUService) Forget(ctx context.Context, query string) error {
	// Mock forget - in real implementation this would use MemU's actual forget
	return nil
}

// MemUMemorizeParams for memorize tool
type MemUMemorizeParams struct {
	ResourceURL string `json:"resource_url" description:"URL or path to resource to memorize (file path, URL, etc.)"`
	Modality    string `json:"modality,omitempty" description:"Type of content (text, image, audio, video, conversation, log)" jsonschema:"enum=text,enum=image,enum=audio,enum=video,enum=conversation,enum=log,default=text"`
}

// MemURetrieveParams for retrieve tool
type MemURetrieveParams struct {
	Queries []string `json:"queries" description:"List of queries to retrieve relevant memories for"`
	Method  string   `json:"method,omitempty" description:"Retrieval method to use" jsonschema:"enum=rag,enum=llm,default=rag"`
}

// MemUSearchParams for search tool
type MemUSearchParams struct {
	Query  string `json:"query" description:"Search query to find specific memories"`
	Method string `json:"method,omitempty" description:"Search method to use" jsonschema:"enum=rag,enum=llm,default=rag"`
}

// MemUForgetParams for forget tool
type MemUForgetParams struct {
	Query string `json:"query" description:"Query pattern for memories to forget"`
}

// MemUMemorizeResponse for memorize tool response
type MemUMemorizeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Items  int    `json:"items_extracted,omitempty"`
}

// MemURetrieveResponse for retrieve tool response
type MemURetrieveResponse struct {
	Memories []string `json:"memories"`
	Count    int      `json:"count"`
	Method   string   `json:"method_used"`
}

// MemUSearchResponse for search tool response
type MemUSearchResponse struct {
	Results []string `json:"results"`
	Count   int     `json:"count"`
	Query   string  `json:"query"`
}

// MemUForgetResponse for forget tool response
type MemUForgetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"memories_removed,omitempty"`
}

type memuTool struct {
	workingDir  string
	permissions permission.Service
	config      *config.MemUConfig
	service     MemUMemoryService
}

// NewMemUTool creates MemU tools
func NewMemUTool(workingDir string, permissions permission.Service, config *config.MemUConfig) []fantasy.AgentTool {
	// If MemU is not configured or disabled, return empty
	if config == nil || !config.Enabled {
		return nil
	}

	// For now, use mock service. In real implementation, this would initialize
	// actual MemU service with proper configuration
	service := &MockMemUService{}

	tool := &memuTool{
		workingDir:  workingDir,
		permissions: permissions,
		config:      config,
		service:     service,
	}

	return []fantasy.AgentTool{
		fantasy.NewAgentTool(
			"memu_memorize",
			string(memuDescription),
			tool.memorize,
		),
		fantasy.NewAgentTool(
			"memu_retrieve",
			"Retrieve relevant memories based on queries using MemU memory framework",
			tool.retrieve,
		),
		fantasy.NewAgentTool(
			"memu_search",
			"Search memories for specific queries using MemU memory framework",
			tool.search,
		),
		fantasy.NewAgentTool(
			"memu_forget",
			"Remove specific memories from MemU memory store",
			tool.forget,
		),
	}
}

func (t *memuTool) memorize(ctx context.Context, params MemUMemorizeParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// Check if it's a local file path
	if strings.HasPrefix(params.ResourceURL, "./") || strings.HasPrefix(params.ResourceURL, "/") || !strings.Contains(params.ResourceURL, "://") {
		// Resolve relative path
		filePath := filepathext.SmartJoin(t.workingDir, params.ResourceURL)

		// Check file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("File not found: %s", filePath)), nil
		}

		params.ResourceURL = filePath
	}

	if err := t.service.Memorize(ctx, params.ResourceURL, params.Modality); err != nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("Failed to memorize: %v", err)), nil
	}

	response := MemUMemorizeResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully memorized: %s", params.ResourceURL),
	}
	
	responseBytes, _ := json.Marshal(response)
	return fantasy.NewTextResponse(string(responseBytes)), nil
}

func (t *memuTool) retrieve(ctx context.Context, params MemURetrieveParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	if len(params.Queries) == 0 {
		response := MemURetrieveResponse{
			Memories: []string{},
			Count:    0,
			Method:   params.Method,
		}
		responseBytes, _ := json.Marshal(response)
		return fantasy.NewTextResponse(string(responseBytes)), nil
	}

	memories, err := t.service.Retrieve(ctx, params.Queries)
	if err != nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("Failed to retrieve memories: %v", err)), nil
	}

	response := MemURetrieveResponse{
		Memories: memories,
		Count:    len(memories),
		Method:   params.Method,
	}
	
	responseBytes, _ := json.Marshal(response)
	return fantasy.NewTextResponse(string(responseBytes)), nil
}

func (t *memuTool) search(ctx context.Context, params MemUSearchParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	if params.Query == "" {
		response := MemUSearchResponse{
			Results: []string{},
			Count:   0,
			Query:   params.Query,
		}
		responseBytes, _ := json.Marshal(response)
		return fantasy.NewTextResponse(string(responseBytes)), nil
	}

	results, err := t.service.Search(ctx, params.Query)
	if err != nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("Failed to search memories: %v", err)), nil
	}

	response := MemUSearchResponse{
		Results: results,
		Count:   len(results),
		Query:   params.Query,
	}
	
	responseBytes, _ := json.Marshal(response)
	return fantasy.NewTextResponse(string(responseBytes)), nil
}

func (t *memuTool) forget(ctx context.Context, params MemUForgetParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	if params.Query == "" {
		return fantasy.NewTextErrorResponse("Query cannot be empty"), nil
	}

	if err := t.service.Forget(ctx, params.Query); err != nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("Failed to forget memories: %v", err)), nil
	}

	response := MemUForgetResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully forgot memories matching: %s", params.Query),
	}
	
	responseBytes, _ := json.Marshal(response)
	return fantasy.NewTextResponse(string(responseBytes)), nil
}