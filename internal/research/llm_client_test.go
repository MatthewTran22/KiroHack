package research

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGeminiLLMClient(t *testing.T) {
	client := NewGeminiLLMClient("test-api-key", "")
	
	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, "gemini-1.5-flash", client.model)
	assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta", client.baseURL)
	assert.NotNil(t, client.httpClient)
}

func TestNewGeminiLLMClient_CustomModel(t *testing.T) {
	client := NewGeminiLLMClient("test-api-key", "gemini-pro")
	
	assert.Equal(t, "gemini-pro", client.model)
}

func TestGeminiLLMClient_GenerateText(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/models/gemini-1.5-flash:generateContent")
		assert.Contains(t, r.URL.RawQuery, "key=test-api-key")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		// Verify request body
		var request GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)
		
		assert.Len(t, request.Contents, 1)
		assert.Len(t, request.Contents[0].Parts, 1)
		assert.Equal(t, "What is healthcare policy?", request.Contents[0].Parts[0].Text)
		assert.NotNil(t, request.GenerationConfig)
		assert.Equal(t, 0.7, *request.GenerationConfig.Temperature)
		assert.Equal(t, 1000, *request.GenerationConfig.MaxOutputTokens)
		assert.Len(t, request.SafetySettings, 4)
		
		// Mock response
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "Healthcare policy refers to decisions, plans, and actions that are undertaken to achieve specific healthcare goals within a society."},
						},
					},
					FinishReason: "STOP",
					SafetyRatings: []GeminiSafetyRating{
						{Category: "HARM_CATEGORY_HARASSMENT", Probability: "NEGLIGIBLE"},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	options := LLMOptions{
		Temperature: 0.7,
		MaxTokens:   1000,
		TopP:        0.9,
	}
	
	ctx := context.Background()
	result, err := client.GenerateText(ctx, "What is healthcare policy?", options)
	
	require.NoError(t, err)
	assert.Equal(t, "Healthcare policy refers to decisions, plans, and actions that are undertaken to achieve specific healthcare goals within a society.", result)
}

func TestGeminiLLMClient_GenerateText_NoOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)
		
		// Should not have generation config when no options provided
		assert.Nil(t, request.GenerationConfig)
		
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "Response without options"},
						},
					},
					FinishReason: "STOP",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	ctx := context.Background()
	result, err := client.GenerateText(ctx, "Test prompt", LLMOptions{})
	
	require.NoError(t, err)
	assert.Equal(t, "Response without options", result)
}

func TestGeminiLLMClient_AnalyzeText_PolicyImpact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)
		
		// Verify the prompt contains policy impact analysis instructions
		prompt := request.Contents[0].Parts[0].Text
		assert.Contains(t, prompt, "policy implications and impacts")
		assert.Contains(t, prompt, "JSON format")
		
		// Mock JSON response
		jsonResponse := `{
			"policy_areas": ["healthcare", "public health"],
			"impacts": {
				"positive": ["improved patient care", "cost reduction"],
				"negative": ["implementation challenges", "resistance to change"]
			},
			"stakeholders": ["patients", "healthcare providers", "government"],
			"urgency": "high",
			"recommended_actions": ["stakeholder engagement", "phased rollout"]
		}`
		
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: jsonResponse},
						},
					},
					FinishReason: "STOP",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	ctx := context.Background()
	result, err := client.AnalyzeText(ctx, "Healthcare policy document content", "policy_impact")
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Verify parsed JSON structure
	policyAreas, ok := result["policy_areas"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, policyAreas, 2)
	assert.Equal(t, "healthcare", policyAreas[0])
	
	impacts, ok := result["impacts"].(map[string]interface{})
	assert.True(t, ok)
	
	positive, ok := impacts["positive"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, positive, 2)
	
	urgency, ok := result["urgency"].(string)
	assert.True(t, ok)
	assert.Equal(t, "high", urgency)
}

func TestGeminiLLMClient_AnalyzeText_Sentiment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)
		
		prompt := request.Contents[0].Parts[0].Text
		assert.Contains(t, prompt, "sentiment and tone")
		assert.Contains(t, prompt, "confidence level")
		
		jsonResponse := `{
			"sentiment": "positive",
			"confidence": 0.85,
			"emotional_indicators": ["optimistic", "supportive"],
			"tone": "formal"
		}`
		
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: jsonResponse},
						},
					},
					FinishReason: "STOP",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	ctx := context.Background()
	result, err := client.AnalyzeText(ctx, "This is a positive policy announcement", "sentiment")
	
	require.NoError(t, err)
	assert.Equal(t, "positive", result["sentiment"])
	assert.Equal(t, 0.85, result["confidence"])
	assert.Equal(t, "formal", result["tone"])
}

func TestGeminiLLMClient_AnalyzeText_UnsupportedType(t *testing.T) {
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	
	ctx := context.Background()
	_, err := client.AnalyzeText(ctx, "test text", "unsupported_type")
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported analysis type")
}

func TestGeminiLLMClient_AnalyzeText_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return invalid JSON
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "This is not valid JSON"},
						},
					},
					FinishReason: "STOP",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	ctx := context.Background()
	result, err := client.AnalyzeText(ctx, "test text", "sentiment")
	
	require.NoError(t, err)
	assert.Equal(t, "sentiment", result["analysis_type"])
	assert.Equal(t, "This is not valid JSON", result["raw_response"])
	assert.Contains(t, result["error"].(string), "failed to parse structured response")
}

func TestGeminiLLMClient_SummarizeText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)
		
		prompt := request.Contents[0].Parts[0].Text
		assert.Contains(t, prompt, "concise summary")
		assert.Contains(t, prompt, "no more than 100 words")
		assert.Contains(t, prompt, "government policy analysis")
		
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "This is a concise summary of the healthcare policy document highlighting key reforms and implementation strategies."},
						},
					},
					FinishReason: "STOP",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	longText := strings.Repeat("Healthcare policy reform is important. ", 50)
	
	ctx := context.Background()
	summary, err := client.SummarizeText(ctx, longText, 100)
	
	require.NoError(t, err)
	assert.Equal(t, "This is a concise summary of the healthcare policy document highlighting key reforms and implementation strategies.", summary)
}

func TestGeminiLLMClient_ExtractKeywords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)
		
		prompt := request.Contents[0].Parts[0].Text
		assert.Contains(t, prompt, "5 most important keywords")
		assert.Contains(t, prompt, "government policy research")
		assert.Contains(t, prompt, "one per line")
		
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "1. healthcare policy\n2. patient care\n3. cost reduction\n4. implementation\n5. stakeholders"},
						},
					},
					FinishReason: "STOP",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	text := "Healthcare policy reform focuses on improving patient care while reducing costs through better implementation strategies involving key stakeholders."
	
	ctx := context.Background()
	keywords, err := client.ExtractKeywords(ctx, text, 5)
	
	require.NoError(t, err)
	assert.Len(t, keywords, 5)
	assert.Equal(t, "healthcare policy", keywords[0])
	assert.Equal(t, "patient care", keywords[1])
	assert.Equal(t, "cost reduction", keywords[2])
	assert.Equal(t, "implementation", keywords[3])
	assert.Equal(t, "stakeholders", keywords[4])
}

func TestGeminiLLMClient_ExtractKeywords_WithNumbering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "• healthcare\n- policy\n1. reform\n2. implementation\n• stakeholders\nextra keyword"},
						},
					},
					FinishReason: "STOP",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	ctx := context.Background()
	keywords, err := client.ExtractKeywords(ctx, "test text", 3)
	
	require.NoError(t, err)
	assert.Len(t, keywords, 3) // Limited to requested number
	assert.Equal(t, "healthcare", keywords[0])
	assert.Equal(t, "policy", keywords[1])
	assert.Equal(t, "reform", keywords[2])
}

func TestGeminiLLMClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		responseBody  interface{}
		statusCode    int
		expectedError string
	}{
		{
			name: "API Error Response",
			responseBody: GeminiResponse{
				Error: &GeminiError{
					Code:    400,
					Message: "Invalid request",
					Status:  "INVALID_ARGUMENT",
				},
			},
			statusCode:    http.StatusBadRequest,
			expectedError: "API error 400: Invalid request",
		},
		{
			name:          "Invalid JSON Response",
			responseBody:  "invalid json",
			statusCode:    http.StatusOK,
			expectedError: "failed to parse response",
		},
		{
			name: "No Candidates",
			responseBody: GeminiResponse{
				Candidates: []GeminiCandidate{},
			},
			statusCode:    http.StatusOK,
			expectedError: "no candidates in response",
		},
		{
			name: "No Parts in Candidate",
			responseBody: GeminiResponse{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Parts: []GeminiPart{},
						},
					},
				},
			},
			statusCode:    http.StatusOK,
			expectedError: "no parts in candidate response",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if str, ok := tt.responseBody.(string); ok {
					w.Write([]byte(str))
				} else {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()
			
			client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
			client.baseURL = server.URL
			
			ctx := context.Background()
			_, err := client.GenerateText(ctx, "test prompt", LLMOptions{})
			
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestGeminiLLMClient_MakeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL contains API key
		assert.Contains(t, r.URL.RawQuery, "key=test-api-key")
		
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "Test response"},
						},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewGeminiLLMClient("test-api-key", "gemini-1.5-flash")
	client.baseURL = server.URL
	
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: "Test prompt"},
				},
			},
		},
	}
	
	ctx := context.Background()
	response, err := client.makeRequest(ctx, request)
	
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Candidates, 1)
	assert.Equal(t, "Test response", response.Candidates[0].Content.Parts[0].Text)
}