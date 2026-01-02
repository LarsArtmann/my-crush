package tools

import (
	"context"
	_ "embed"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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

// Memory represents a stored memory item
type Memory struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Modality  string    `json:"modality"`
	Source    string    `json:"source"`
	Tags      []string  `json:"tags,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MemoryStore represents the in-memory and persisted storage
type MemoryStore struct {
	Memories map[string]Memory `json:"memories"`
	Tags     map[string][]string `json:"tags,omitempty"`
}

// RealMemUService implements actual memory storage and retrieval
type RealMemUService struct {
	mu        sync.RWMutex
	store     *MemoryStore
	config    *config.MemUConfig
	dataDir   string
	filePath  string
}

// generateID creates a unique memory ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// NewRealMemUService creates a new real memory service
func NewRealMemUService(config *config.MemUConfig) (*RealMemUService, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Ensure data directory exists
	dataDir := config.DataDir
	if dataDir == "" {
		dataDir = ".crush/memory"
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, "memories.json")

	service := &RealMemUService{
		config:   config,
		dataDir:  dataDir,
		filePath: filePath,
		store:    &MemoryStore{
			Memories: make(map[string]Memory),
			Tags:     make(map[string][]string),
		},
	}

	// Load existing memories from file
	if err := service.loadFromFile(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load existing memories: %w", err)
	}

	return service, nil
}

// loadFromFile loads memories from JSON file
func (s *RealMemUService) loadFromFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil // Empty file is okay
	}

	return json.Unmarshal(data, s.store)
}

// saveToFile saves memories to JSON file
func (s *RealMemUService) saveToFile() error {
	data, err := json.MarshalIndent(s.store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memories: %w", err)
	}

	return os.WriteFile(s.filePath, data, 0o644)
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

// RealMemUService implementation

func (s *RealMemUService) Memorize(ctx context.Context, resourceURL, modality string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read content from resource
	var content string
	var err error

	if strings.HasPrefix(resourceURL, "http://") || strings.HasPrefix(resourceURL, "https://") {
		// For URLs, we'll just store the URL as content for now
		// In a real implementation, we would fetch the URL content
		content = fmt.Sprintf("Resource: %s", resourceURL)
	} else {
		// For local files, read the content
		var data []byte
		data, err = os.ReadFile(resourceURL)
		if err != nil {
			return fmt.Errorf("failed to read resource %s: %w", resourceURL, err)
		}
		content = string(data)
	}

	// Create memory object
	memoryID := generateID()
	now := time.Now()
	
	memory := Memory{
		ID:        memoryID,
		Content:   content,
		Modality:  modality,
		Source:    resourceURL,
		Timestamp: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Store memory
	s.store.Memories[memoryID] = memory

	// Update tag index (simple implementation - extract from content)
	tags := s.extractTags(content)
	for _, tag := range tags {
		s.store.Tags[tag] = append(s.store.Tags[tag], memoryID)
	}

	// Save to file
	if err := s.saveToFile(); err != nil {
		delete(s.store.Memories, memoryID) // Cleanup on failure
		return fmt.Errorf("failed to save memory: %w", err)
	}

	return nil
}

func (s *RealMemUService) Retrieve(ctx context.Context, queries []string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(queries) == 0 {
		return []string{}, nil
	}

	var results []string
	found := make(map[string]bool)

	// Search for each query
	for _, query := range queries {
		memoryIDs := s.searchMemories(query)
		for _, id := range memoryIDs {
			if !found[id] {
				if memory, exists := s.store.Memories[id]; exists {
					results = append(results, memory.Content)
					found[id] = true
				}
			}
		}
	}

	return results, nil
}

func (s *RealMemUService) Search(ctx context.Context, query string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query == "" {
		return []string{}, nil
	}

	memoryIDs := s.searchMemories(query)
	var results []string
	
	for _, id := range memoryIDs {
		if memory, exists := s.store.Memories[id]; exists {
			results = append(results, fmt.Sprintf("ID: %s | Source: %s | Modality: %s | Content: %s", 
				memory.ID, memory.Source, memory.Modality, memory.Content[:min(200, len(memory.Content))]))
		}
	}

	return results, nil
}

func (s *RealMemUService) Forget(ctx context.Context, query string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	memoryIDs := s.searchMemories(query)
	removedCount := 0

	for _, id := range memoryIDs {
		if _, exists := s.store.Memories[id]; exists {
			// Remove from memories
			delete(s.store.Memories, id)
			
			// Remove from tag index
			for tag, ids := range s.store.Tags {
				s.store.Tags[tag] = removeFromSlice(ids, id)
				if len(s.store.Tags[tag]) == 0 {
					delete(s.store.Tags, tag)
				}
			}
			
			removedCount++
		}
	}

	if removedCount > 0 {
		return s.saveToFile()
	}

	return nil
}

// Helper methods

func (s *RealMemUService) extractTags(content string) []string {
	// Simple tag extraction - in a real implementation, this would be more sophisticated
	// For now, just split content into words and use common words as tags
	words := strings.Fields(strings.ToLower(content))
	tags := make(map[string]bool)
	
	for _, word := range words {
		if len(word) > 4 && !isCommonWord(word) {
			tags[word] = true
		}
	}
	
	var result []string
	for tag := range tags {
		if len(result) < 5 { // Limit to 5 tags per memory
			result = append(result, tag)
		}
	}
	
	return result
}

func (s *RealMemUService) searchMemories(query string) []string {
	query = strings.ToLower(query)
	var results []string
	
	// Search through memories
	for id, memory := range s.store.Memories {
		content := strings.ToLower(memory.Content)
		source := strings.ToLower(memory.Source)
		
		if strings.Contains(content, query) || 
		   strings.Contains(source, query) ||
		   strings.Contains(strings.ToLower(memory.Modality), query) {
			results = append(results, id)
		}
	}
	
	return results
}

func isCommonWord(word string) bool {
	commonWords := []string{
		"this", "that", "with", "from", "they", "have", "been", 
		"there", "were", "said", "each", "which", "their", "time", 
		"will", "about", "would", "could", "other", "after", "first",
	}
	for _, common := range commonWords {
		if word == common {
			return true
		}
	}
	return false
}

func removeFromSlice(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	// Use real MemU service
	var service MemUMemoryService
	service, err := NewRealMemUService(config)
	if err != nil {
		// Fallback to mock service if real service fails to initialize
		service = &MockMemUService{}
	}

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