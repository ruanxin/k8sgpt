package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

// ChatCompletionRequest represents a request structure for chat completion API.
type ChatCompletionRequest struct {
	DeploymentId     string                         `json:"deployment_id"`
	Messages         []openai.ChatCompletionMessage `json:"messages"`
	MaxTokens        int                            `json:"max_tokens,omitempty"`
	Temperature      float32                        `json:"temperature,omitempty"`
	TopP             float32                        `json:"top_p,omitempty"`
	N                int                            `json:"n,omitempty"`
	Stream           bool                           `json:"stream,omitempty"`
	Stop             []string                       `json:"stop,omitempty"`
	PresencePenalty  float32                        `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32                        `json:"frequency_penalty,omitempty"`
	// LogitBias is must be a token id string (specified by their token ID in the tokenizer), not a word string.
	// incorrect: `"logit_bias":{"You": 6}`, correct: `"logit_bias":{"1639": 6}`
	// refs: https://platform.openai.com/docs/api-reference/chat/create#chat/create-logit_bias
	LogitBias    map[string]int `json:"logit_bias,omitempty"`
	User         string         `json:"user,omitempty"`
	FunctionCall any            `json:"function_call,omitempty"`
}

func isFailureStatusCode(resp *http.Response) bool {
	return resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest
}

func decodeResponse(body io.Reader, v any) error {
	if v == nil {
		return nil
	}

	if result, ok := v.(*string); ok {
		return decodeString(body, result)
	}
	return json.NewDecoder(body).Decode(v)
}

func handleErrorResp(resp *http.Response) error {
	var errRes openai.ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&errRes)
	if err != nil || errRes.Error == nil {
		reqErr := &openai.RequestError{
			HTTPStatusCode: resp.StatusCode,
			Err:            err,
		}
		if errRes.Error != nil {
			reqErr.Err = errRes.Error
		}
		return reqErr
	}

	errRes.Error.HTTPStatusCode = resp.StatusCode
	return errRes.Error
}

func decodeString(body io.Reader, output *string) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	*output = string(b)
	return nil
}

func build(
	ctx context.Context,
	method string,
	url string,
	body any,
	header http.Header,
) (req *http.Request, err error) {
	var bodyReader io.Reader
	if body != nil {
		if v, ok := body.(io.Reader); ok {
			bodyReader = v
		} else {
			var reqBytes []byte
			reqBytes, err = json.Marshal(body)
			if err != nil {
				return
			}
			bodyReader = bytes.NewBuffer(reqBytes)
		}
	}
	req, err = http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return
	}
	if header != nil {
		req.Header = header
	}
	return
}

type requestOptions struct {
	body   any
	header http.Header
}

type requestOption func(*requestOptions)

func withBody(body any) requestOption {
	return func(args *requestOptions) {
		args.body = body
	}
}
