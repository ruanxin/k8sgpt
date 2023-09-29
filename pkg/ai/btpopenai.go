package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/k8sgpt-ai/k8sgpt/pkg/cache"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"

	"github.com/fatih/color"

	"github.com/sashabaranov/go-openai"
)

type BTPAIClient struct {
	HTTPClient  *http.Client
	language    string
	model       string
	temperature float32
	baseURL     string
	token       string
}

func (c *BTPAIClient) Configure(config IAIConfig, lang string) error {
	token := config.GetPassword()
	baseURL := config.GetBaseURL()
	c.language = lang
	c.model = config.GetModel()
	c.temperature = config.GetTemperature()
	c.baseURL = baseURL
	c.token = token
	c.HTTPClient = &http.Client{}
	return nil
}

func (c *BTPAIClient) GetCompletion(ctx context.Context, prompt string, promptTmpl string) (string, error) {
	// Create a completion request
	content := fmt.Sprintf(default_prompt, c.language, prompt)
	resp, err := c.createChatCompletion(ctx, ChatCompletionRequest{
		DeploymentId: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: content,
			},
		},
		Temperature: c.temperature,
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *BTPAIClient) createChatCompletion(ctx context.Context, request ChatCompletionRequest) (response openai.ChatCompletionResponse, err error) {
	if request.Stream {
		err = openai.ErrChatCompletionStreamNotSupported
		return
	}

	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("%s%s", c.baseURL, "/api/v1/completions"), withBody(request))
	if err != nil {
		return
	}

	err = c.sendRequest(req, &response)
	return
}

func (c *BTPAIClient) newRequest(ctx context.Context, method, url string, setters ...requestOption) (*http.Request, error) {
	// Default Options
	args := &requestOptions{
		body:   nil,
		header: make(http.Header),
	}
	for _, setter := range setters {
		setter(args)
	}
	req, err := build(ctx, method, url, args.body, args.header)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *BTPAIClient) sendRequest(req *http.Request, v any) error {
	req.Header.Set("Accept", "application/json; charset=utf-8")

	// Check whether Content-Type is already set, Upload Files API requires
	// Content-Type == multipart/form-data
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if isFailureStatusCode(res) {
		return handleErrorResp(res)
	}

	return decodeResponse(res.Body, v)
}

func (a *BTPAIClient) Parse(ctx context.Context, prompt []string, cache cache.ICache, promptTmpl string) (string, error) {
	inputKey := strings.Join(prompt, " ")
	// Check for cached data
	cacheKey := util.GetCacheKey(a.GetName(), a.language, inputKey)

	if !cache.IsCacheDisabled() && cache.Exists(cacheKey) {
		response, err := cache.Load(cacheKey)
		if err != nil {
			return "", err
		}

		if response != "" {
			output, err := base64.StdEncoding.DecodeString(response)
			if err != nil {
				color.Red("error decoding cached data: %v", err)
				return "", nil
			}
			return string(output), nil
		}
	}

	response, err := a.GetCompletion(ctx, inputKey, promptTmpl)
	if err != nil {
		return "", err
	}

	err = cache.Store(cacheKey, base64.StdEncoding.EncodeToString([]byte(response)))

	if err != nil {
		color.Red("error storing value to cache: %v", err)
		return "", nil
	}

	return response, nil
}

func (a *BTPAIClient) GetName() string {
	return "btpopenai"
}
