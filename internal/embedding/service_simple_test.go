package embedding

import (
	"testing"

	"ai-government-consultant/pkg/logger"
)

func TestNewServiceSimple(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				GeminiAPIKey: "test-api-key",
				Logger:       logger.NewTestLogger(),
			},
			expectError: false,
		},
		{
			name: "missing API key",
			config: &Config{
				Logger: logger.NewTestLogger(),
			},
			expectError: true,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if service == nil {
				t.Error("expected service but got nil")
			}
		})
	}
}

func TestSearchOptions(t *testing.T) {
	options := &SearchOptions{
		Limit:      10,
		Threshold:  0.7,
		Collection: "documents",
		Filters:    map[string]interface{}{"category": "policy"},
	}

	if options.Limit != 10 {
		t.Errorf("expected limit 10, got %d", options.Limit)
	}
	if options.Threshold != 0.7 {
		t.Errorf("expected threshold 0.7, got %f", options.Threshold)
	}
	if options.Collection != "documents" {
		t.Errorf("expected collection 'documents', got %s", options.Collection)
	}
	if len(options.Filters) != 1 {
		t.Errorf("expected 1 filter, got %d", len(options.Filters))
	}
}

func TestSearchResult(t *testing.T) {
	result := SearchResult{
		ID:    "test-id",
		Score: 0.85,
		Metadata: map[string]interface{}{
			"type": "document",
		},
	}

	if result.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %s", result.ID)
	}
	if result.Score != 0.85 {
		t.Errorf("expected score 0.85, got %f", result.Score)
	}
	if result.Metadata["type"] != "document" {
		t.Errorf("expected metadata type 'document', got %v", result.Metadata["type"])
	}
}
