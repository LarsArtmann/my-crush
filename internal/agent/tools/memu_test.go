package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
)

func TestRealMemUService(t *testing.T) {
	t.Parallel()
	
	tmpDir := t.TempDir()
	
	// Create a test config
	cfg := &config.MemUConfig{
		Enabled: true,
		DataDir: tmpDir,
	}
	
	// Create service
	service, err := NewRealMemUService(cfg)
	if err != nil {
		t.Fatalf("Failed to create RealMemUService: %v", err)
	}
	
	ctx := context.Background()
	
	// Test 1: Memorize a simple text
	testContent := "This is a test memory about AI and machine learning"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	if err := service.Memorize(ctx, testFile, "text"); err != nil {
		t.Fatalf("Failed to memorize: %v", err)
	}
	
	// Test 2: Search for the memory
	results, err := service.Search(ctx, "machine learning")
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}
	
	// Test 3: Retrieve memories
	memories, err := service.Retrieve(ctx, []string{"AI", "machine learning"})
	if err != nil {
		t.Fatalf("Failed to retrieve: %v", err)
	}
	
	if len(memories) == 0 {
		t.Fatal("Expected at least one memory")
	}
	
	// Test 4: Test persistence by creating a new service instance
	service2, err := NewRealMemUService(cfg)
	if err != nil {
		t.Fatalf("Failed to create second RealMemUService: %v", err)
	}
	
	// Check that memories are still there
	results2, err := service2.Search(ctx, "machine learning")
	if err != nil {
		t.Fatalf("Failed to search with second service: %v", err)
	}
	
	if len(results2) == 0 {
		t.Fatal("Expected persisted memories in second service")
	}
	
	// Test 5: Forget memories
	if err := service2.Forget(ctx, "test memory"); err != nil {
		t.Fatalf("Failed to forget: %v", err)
	}
	
	// Verify memories are gone
	results3, err := service2.Search(ctx, "test memory")
	if err != nil {
		t.Fatalf("Failed to search after forget: %v", err)
	}
	
	if len(results3) > 0 {
		t.Fatal("Expected no results after forget")
	}
}

func TestNewMemUToolWithRealService(t *testing.T) {
	t.Parallel()
	
	tmpDir := t.TempDir()
	cfg := &config.MemUConfig{
		Enabled: true,
		DataDir: tmpDir,
	}
	
	// Test tool creation with real service
	tools := NewMemUTool("/test", nil, cfg)
	if tools == nil {
		t.Fatal("Expected tools to be created")
	}
	
	if len(tools) != 4 {
		t.Fatalf("Expected 4 tools, got %d", len(tools))
	}
}

func TestNewMemUToolWithDisabledConfig(t *testing.T) {
	t.Parallel()
	
	cfg := &config.MemUConfig{
		Enabled: false,
	}
	
	// Test tool creation with disabled config
	tools := NewMemUTool("/test", nil, cfg)
	if tools != nil {
		t.Fatal("Expected no tools when disabled")
	}
}

func TestNewMemUToolWithNilConfig(t *testing.T) {
	t.Parallel()
	
	// Test tool creation with nil config
	tools := NewMemUTool("/test", nil, nil)
	if tools != nil {
		t.Fatal("Expected no tools with nil config")
	}
}