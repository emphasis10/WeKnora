package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// RemoteAPIChat 实现了基于的聊天
type RemoteAPIChat struct {
	modelName string
	client    openai.Client
	modelID   string
	baseURL   string
	apiKey    string
}

// QwenChatCompletionRequest 用于 qwen 模型的自定义请求结构体
type QwenChatCompletionRequest struct {
	Model               string                 `json:"model"`
	Messages            []map[string]any       `json:"messages"`
	Stream              bool                   `json:"stream,omitempty"`
	Temperature         float32                `json:"temperature,omitempty"`
	TopP                float32                `json:"top_p,omitempty"`
	MaxTokens           int                    `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                    `json:"max_completion_tokens,omitempty"`
	FrequencyPenalty    float32                `json:"frequency_penalty,omitempty"`
	PresencePenalty     float32                `json:"presence_penalty,omitempty"`
	Tools               []map[string]any       `json:"tools,omitempty"`
	ToolChoice          any                    `json:"tool_choice,omitempty"`
	ChatTemplateKwargs  map[string]interface{} `json:"chat_template_kwargs,omitempty"`
	EnableThinking      *bool                  `json:"enable_thinking,omitempty"` // qwen 模型专用字段
}

// NewRemoteAPIChat 调用远程API 聊天实例
func NewRemoteAPIChat(chatConfig *ChatConfig) (*RemoteAPIChat, error) {
	apiKey := chatConfig.APIKey
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if chatConfig.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(chatConfig.BaseURL))
	}
	return &RemoteAPIChat{
		modelName: chatConfig.ModelName,
		client:    openai.NewClient(opts...),
		modelID:   chatConfig.ModelID,
		baseURL:   chatConfig.BaseURL,
		apiKey:    apiKey,
	}, nil
}

// convertMessages 转换消息格式为OpenAI格式
func (c *RemoteAPIChat) convertMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			openaiMessages = append(openaiMessages, openai.SystemMessage(msg.Content))
		case "user":
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				// For assistant messages with tool calls, we need to use raw HTTP
				// as the SDK doesn't provide a simple way to construct these
				openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
			} else {
				openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
			}
		case "tool":
			openaiMessages = append(openaiMessages, openai.ToolMessage(msg.ToolCallID, msg.Content))
		default:
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		}
	}
	return openaiMessages
}

// convertMessagesToRaw 转换消息格式为原始 map 格式（用于自定义请求）
func (c *RemoteAPIChat) convertMessagesToRaw(messages []Message) []map[string]any {
	rawMessages := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		rawMsg := map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]any, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			rawMsg["tool_calls"] = toolCalls
		}
		if msg.ToolCallID != "" {
			rawMsg["tool_call_id"] = msg.ToolCallID
		}
		if msg.Name != "" {
			rawMsg["name"] = msg.Name
		}
		rawMessages = append(rawMessages, rawMsg)
	}
	return rawMessages
}

// isQwenModel 检查是否为 qwen 模型
func (c *RemoteAPIChat) isAliyunQwen3Model() bool {
	return strings.HasPrefix(c.modelName, "qwen3-") && c.baseURL == "https://dashscope.aliyuncs.com/compatible-mode/v1"
}

// isDeepSeekModel 检查是否为 DeepSeek 模型
func (c *RemoteAPIChat) isDeepSeekModel() bool {
	return strings.Contains(strings.ToLower(c.modelName), "deepseek")
}

// hasToolCalls 检查消息中是否有 tool calls
func (c *RemoteAPIChat) hasToolCalls(messages []Message) bool {
	for _, msg := range messages {
		if len(msg.ToolCalls) > 0 {
			return true
		}
	}
	return false
}

// buildChatCompletionRequest 构建聊天请求参数
func (c *RemoteAPIChat) buildChatCompletionRequest(messages []Message, opts *ChatOptions) openai.ChatCompletionNewParams {
	req := openai.ChatCompletionNewParams{
		Model:    c.modelName,
		Messages: c.convertMessages(messages),
	}

	// 添加可选参数
	if opts != nil {
		if opts.Temperature > 0 {
			req.Temperature = openai.Float(opts.Temperature)
		}
		if opts.TopP > 0 {
			req.TopP = openai.Float(opts.TopP)
		}
		if opts.MaxTokens > 0 {
			req.MaxTokens = openai.Int(int64(opts.MaxTokens))
		}
		if opts.MaxCompletionTokens > 0 {
			req.MaxCompletionTokens = openai.Int(int64(opts.MaxCompletionTokens))
		}
		if opts.FrequencyPenalty > 0 {
			req.FrequencyPenalty = openai.Float(opts.FrequencyPenalty)
		}
		if opts.PresencePenalty > 0 {
			req.PresencePenalty = openai.Float(opts.PresencePenalty)
		}

		// 处理 Tools（函数定义）
		if len(opts.Tools) > 0 {
			tools := make([]openai.ChatCompletionToolUnionParam, 0, len(opts.Tools))
			for _, tool := range opts.Tools {
				openaiTool := openai.ChatCompletionToolUnionParam{
					OfFunction: &openai.ChatCompletionFunctionToolParam{
						Function: openai.FunctionDefinitionParam{
							Name:        tool.Function.Name,
							Description: openai.String(tool.Function.Description),
						},
					},
				}
				// 转换 Parameters
				if tool.Function.Parameters != nil {
					openaiTool.OfFunction.Function.Parameters = openai.FunctionParameters(tool.Function.Parameters)
				}
				tools = append(tools, openaiTool)
			}
			req.Tools = tools
		}

		// 处理 ToolChoice - 使用字符串形式
		if opts.ToolChoice != "" && !c.isDeepSeekModel() {
			// For standard choice options, use OfAuto
			// Note: specific tool name selection is handled via raw HTTP in chatWithRawHTTP
			switch opts.ToolChoice {
			case "none", "required", "auto":
				req.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
					OfAuto: openai.String(opts.ToolChoice),
				}
			}
		}
	}

	return req
}

// buildQwenChatCompletionRequest 构建 qwen 模型的自定义请求
func (c *RemoteAPIChat) buildQwenChatCompletionRequest(messages []Message, opts *ChatOptions, isStream bool) QwenChatCompletionRequest {
	req := QwenChatCompletionRequest{
		Model:    c.modelName,
		Messages: c.convertMessagesToRaw(messages),
		Stream:   isStream,
	}

	thinking := false

	if opts != nil {
		if opts.Temperature > 0 {
			req.Temperature = float32(opts.Temperature)
		}
		if opts.TopP > 0 {
			req.TopP = float32(opts.TopP)
		}
		if opts.MaxTokens > 0 {
			req.MaxTokens = opts.MaxTokens
		}
		if opts.MaxCompletionTokens > 0 {
			req.MaxCompletionTokens = opts.MaxCompletionTokens
		}
		if opts.FrequencyPenalty > 0 {
			req.FrequencyPenalty = float32(opts.FrequencyPenalty)
		}
		if opts.PresencePenalty > 0 {
			req.PresencePenalty = float32(opts.PresencePenalty)
		}
		if opts.Thinking != nil {
			thinking = *opts.Thinking
		}

		// 处理 Tools
		if len(opts.Tools) > 0 {
			tools := make([]map[string]any, 0, len(opts.Tools))
			for _, tool := range opts.Tools {
				tools = append(tools, map[string]any{
					"type": tool.Type,
					"function": map[string]any{
						"name":        tool.Function.Name,
						"description": tool.Function.Description,
						"parameters":  tool.Function.Parameters,
					},
				})
			}
			req.Tools = tools
		}

		// 处理 ToolChoice
		if opts.ToolChoice != "" && !c.isDeepSeekModel() {
			switch opts.ToolChoice {
			case "none", "required", "auto":
				req.ToolChoice = opts.ToolChoice
			default:
				req.ToolChoice = map[string]any{
					"type": "function",
					"function": map[string]any{
						"name": opts.ToolChoice,
					},
				}
			}
		}
	}

	req.ChatTemplateKwargs = map[string]interface{}{
		"enable_thinking": thinking,
	}

	// 对于 qwen 模型，在非流式调用中强制设置 enable_thinking: false
	if !isStream {
		enableThinking := false
		req.EnableThinking = &enableThinking
	}

	return req
}

// Chat 进行非流式聊天
func (c *RemoteAPIChat) Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*types.ChatResponse, error) {
	// 如果是 qwen 模型或者有 tool calls，使用自定义请求（因为 SDK 对 tool calls 支持复杂）
	if c.isAliyunQwen3Model() || c.hasToolCalls(messages) {
		return c.chatWithRawHTTP(ctx, messages, opts)
	}

	// 构建请求参数
	req := c.buildChatCompletionRequest(messages, opts)

	// 发送请求
	resp, err := c.client.Chat.Completions.New(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	choice := resp.Choices[0]
	response := &types.ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
	}

	// 转换 Tool Calls
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]types.LLMToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			response.ToolCalls = append(response.ToolCalls, types.LLMToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: types.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return response, nil
}

// chatWithRawHTTP 使用原始 HTTP 请求处理聊天（用于 qwen 模型和复杂的 tool calls 场景）
func (c *RemoteAPIChat) chatWithRawHTTP(
	ctx context.Context,
	messages []Message,
	opts *ChatOptions,
) (*types.ChatResponse, error) {
	// 构建请求参数
	req := c.buildQwenChatCompletionRequest(messages, opts, false)

	// 序列化请求
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 构建 URL
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := baseURL + "/chat/completions"

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	choice := chatResp.Choices[0]
	response := &types.ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}

	// 转换 Tool Calls
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]types.LLMToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			response.ToolCalls = append(response.ToolCalls, types.LLMToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: types.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return response, nil
}

// ChatStream 进行流式聊天
func (c *RemoteAPIChat) ChatStream(ctx context.Context,
	messages []Message, opts *ChatOptions,
) (<-chan types.StreamResponse, error) {
	// 构建请求参数
	req := c.buildChatCompletionRequest(messages, opts)

	// 创建流式响应通道
	streamChan := make(chan types.StreamResponse)

	// 启动流式请求
	stream := c.client.Chat.Completions.NewStreaming(ctx, req)

	// 在后台处理流式响应
	go func() {
		defer close(streamChan)
		defer stream.Close()

		toolCallMap := make(map[int]*types.LLMToolCall)
		lastFunctionName := make(map[int]string)
		nameNotified := make(map[int]bool)

		buildOrderedToolCalls := func() []types.LLMToolCall {
			if len(toolCallMap) == 0 {
				return nil
			}
			result := make([]types.LLMToolCall, 0, len(toolCallMap))
			for i := 0; i < len(toolCallMap); i++ {
				if tc, ok := toolCallMap[i]; ok && tc != nil {
					result = append(result, *tc)
				}
			}
			if len(result) == 0 {
				return nil
			}
			return result
		}

		for stream.Next() {
			response := stream.Current()

			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta
				isDone := string(response.Choices[0].FinishReason) != ""

				// 收集 tool calls（流式响应中 tool calls 可能分多次返回）
				if len(delta.ToolCalls) > 0 {
					for _, tc := range delta.ToolCalls {
						// 检查是否已经存在该 tool call（通过 index）
						toolCallIndex := int(tc.Index)
						toolCallEntry, exists := toolCallMap[toolCallIndex]
						if !exists || toolCallEntry == nil {
							toolCallEntry = &types.LLMToolCall{
								Type: string(tc.Type),
								Function: types.FunctionCall{
									Name:      "",
									Arguments: "",
								},
							}
							toolCallMap[toolCallIndex] = toolCallEntry
						}

						// 更新 ID、类型
						if tc.ID != "" {
							toolCallEntry.ID = tc.ID
						}
						if tc.Type != "" {
							toolCallEntry.Type = string(tc.Type)
						}

						// 累积函数名称（可能分多次返回）
						if tc.Function.Name != "" {
							toolCallEntry.Function.Name += tc.Function.Name
						}

						// 累积参数（可能为部分 JSON）
						argsUpdated := false
						if tc.Function.Arguments != "" {
							toolCallEntry.Function.Arguments += tc.Function.Arguments
							argsUpdated = true
						}

						currName := toolCallEntry.Function.Name
						if currName != "" &&
							currName == lastFunctionName[toolCallIndex] &&
							argsUpdated &&
							!nameNotified[toolCallIndex] &&
							toolCallEntry.ID != "" {
							streamChan <- types.StreamResponse{
								ResponseType: types.ResponseTypeToolCall,
								Content:      "",
								Done:         false,
								Data: map[string]interface{}{
									"tool_name":    currName,
									"tool_call_id": toolCallEntry.ID,
								},
							}
							nameNotified[toolCallIndex] = true
						}

						lastFunctionName[toolCallIndex] = currName
					}
				}

				// 发送内容块
				if delta.Content != "" {
					streamChan <- types.StreamResponse{
						ResponseType: types.ResponseTypeAnswer,
						Content:      delta.Content,
						Done:         isDone,
						ToolCalls:    buildOrderedToolCalls(),
					}
				}

				// 如果是最后一次响应，确保发送包含所有 tool calls 的响应
				if isDone && len(toolCallMap) > 0 {
					streamChan <- types.StreamResponse{
						ResponseType: types.ResponseTypeAnswer,
						Content:      "",
						Done:         true,
						ToolCalls:    buildOrderedToolCalls(),
					}
				}
			}
		}

		// Check for errors
		if err := stream.Err(); err != nil {
			logger.GetLogger(ctx).Errorf("stream error: %v", err)
		}

		// 发送最后一个响应，包含收集到的 tool calls
		streamChan <- types.StreamResponse{
			ResponseType: types.ResponseTypeAnswer,
			Content:      "",
			Done:         true,
			ToolCalls:    buildOrderedToolCalls(),
		}
	}()

	return streamChan, nil
}

// GetModelName 获取模型名称
func (c *RemoteAPIChat) GetModelName() string {
	return c.modelName
}

// GetModelID 获取模型ID
func (c *RemoteAPIChat) GetModelID() string {
	return c.modelID
}
