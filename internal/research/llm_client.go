package research

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GeminiRequest represents a request to the Gemini API
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings []GeminiSafetySetting `json:"safetySettings,omitempty"`
}

// GeminiContent represents content in a Gemini request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a part of Gemini content
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig represents generation configuration for Gemini
type GeminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// GeminiSafetySetting represents safety settings for Gemini
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse represents a response from the Gemini API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	Error      *GeminiError      `json:"error,omitempty"`
}

// GeminiCandidate represents a candidate response from Gemini
type GeminiCandidate struct {
	Content       GeminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"`
}

// GeminiSafetyRating represents a safety rating from Gemini
type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GeminiError represents an error from the Gemini API
type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// GeminiLLMClient implements LLMClient using Google's Gemini API
type GeminiLLMClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewGeminiLLMClient creates a new Gemini LLM client
func NewGeminiLLMClient(apiKey, model string) *GeminiLLMClient {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	
	return &GeminiLLMClient{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateText generates text based on a prompt
func (c *GeminiLLMClient) GenerateText(ctx context.Context, prompt string, options LLMOptions) (string, error) {
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}
	
	// Apply options
	if options.Temperature > 0 || options.MaxTokens > 0 || options.TopP > 0 {
		config := &GeminiGenerationConfig{}
		
		if options.Temperature > 0 {
			config.Temperature = &options.Temperature
		}
		
		if options.TopP > 0 {
			config.TopP = &options.TopP
		}
		
		if options.MaxTokens > 0 {
			config.MaxOutputTokens = &options.MaxTokens
		}
		
		request.GenerationConfig = config
	}
	
	// Add safety settings for government use
	request.SafetySettings = []GeminiSafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
	}
	
	response, err := c.makeRequest(ctx, request)
	if err != nil {
		return "", err
	}
	
	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}
	
	if len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no parts in candidate response")
	}
	
	return response.Candidates[0].Content.Parts[0].Text, nil
}

// AnalyzeText analyzes text and extracts insights
func (c *GeminiLLMClient) AnalyzeText(ctx context.Context, text string, analysisType string) (map[string]interface{}, error) {
	var prompt string
	
	switch analysisType {
	case "policy_impact":
		prompt = fmt.Sprintf(`Analyze the following text for policy implications and impacts. 
		Provide a structured analysis including:
		1. Key policy areas affected
		2. Potential impacts (positive and negative)
		3. Stakeholders involved
		4. Urgency level
		5. Recommended actions
		
		Text to analyze:
		%s
		
		Please provide the response in JSON format with the following structure:
		{
			"policy_areas": ["area1", "area2"],
			"impacts": {
				"positive": ["impact1", "impact2"],
				"negative": ["impact1", "impact2"]
			},
			"stakeholders": ["stakeholder1", "stakeholder2"],
			"urgency": "low|medium|high|critical",
			"recommended_actions": ["action1", "action2"]
		}`, text)
		
	case "sentiment":
		prompt = fmt.Sprintf(`Analyze the sentiment and tone of the following text. 
		Provide scores for sentiment (positive/negative), confidence level, and key emotional indicators.
		
		Text to analyze:
		%s
		
		Please provide the response in JSON format:
		{
			"sentiment": "positive|negative|neutral",
			"confidence": 0.85,
			"emotional_indicators": ["indicator1", "indicator2"],
			"tone": "formal|informal|urgent|calm"
		}`, text)
		
	case "key_entities":
		prompt = fmt.Sprintf(`Extract key entities from the following text including:
		1. Organizations
		2. People
		3. Locations
		4. Policies/Laws
		5. Dates
		6. Financial amounts
		
		Text to analyze:
		%s
		
		Please provide the response in JSON format:
		{
			"organizations": ["org1", "org2"],
			"people": ["person1", "person2"],
			"locations": ["location1", "location2"],
			"policies": ["policy1", "policy2"],
			"dates": ["date1", "date2"],
			"financial_amounts": ["amount1", "amount2"]
		}`, text)
		
	default:
		return nil, fmt.Errorf("unsupported analysis type: %s", analysisType)
	}
	
	response, err := c.GenerateText(ctx, prompt, LLMOptions{
		Temperature: 0.3, // Lower temperature for more consistent analysis
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to analyze text: %w", err)
	}
	
	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If JSON parsing fails, return a simple text analysis
		return map[string]interface{}{
			"analysis_type": analysisType,
			"raw_response": response,
			"error":        "failed to parse structured response",
		}, nil
	}
	
	return result, nil
}

// SummarizeText creates a summary of the provided text
func (c *GeminiLLMClient) SummarizeText(ctx context.Context, text string, maxLength int) (string, error) {
	prompt := fmt.Sprintf(`Please provide a concise summary of the following text. 
	The summary should be no more than %d words and should capture the key points and main ideas.
	Focus on the most important information that would be relevant for government policy analysis.
	
	Text to summarize:
	%s`, maxLength, text)
	
	return c.GenerateText(ctx, prompt, LLMOptions{
		Temperature: 0.3,
		MaxTokens:   maxLength * 2, // Allow some buffer for token estimation
	})
}

// ExtractKeywords extracts keywords from text
func (c *GeminiLLMClient) ExtractKeywords(ctx context.Context, text string, maxKeywords int) ([]string, error) {
	prompt := fmt.Sprintf(`Extract the %d most important keywords and phrases from the following text. 
	Focus on terms that would be relevant for government policy research and analysis.
	Return only the keywords, one per line, without numbering or additional formatting.
	
	Text to analyze:
	%s`, maxKeywords, text)
	
	response, err := c.GenerateText(ctx, prompt, LLMOptions{
		Temperature: 0.2,
		MaxTokens:   500,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to extract keywords: %w", err)
	}
	
	// Parse keywords from response
	lines := strings.Split(strings.TrimSpace(response), "\n")
	keywords := make([]string, 0, len(lines))
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Remove any numbering or bullet points
			line = strings.TrimLeft(line, "0123456789.-• ")
			if line != "" {
				keywords = append(keywords, line)
			}
		}
	}
	
	// Limit to requested number of keywords
	if len(keywords) > maxKeywords {
		keywords = keywords[:maxKeywords]
	}
	
	return keywords, nil
}

// makeRequest makes a request to the Gemini API
func (c *GeminiLLMClient) makeRequest(ctx context.Context, request GeminiRequest) (*GeminiResponse, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	var response GeminiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("API error %d: %s", response.Error.Code, response.Error.Message)
	}
	
	return &response, nil
}