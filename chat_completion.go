package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Role int

const (
	System Role = iota
	User
	Assistant
)

type ChatMessage struct {
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
	Role      Role   `json:"role"`
}

type ChatCompletion struct {
	client *openai.Client
	store  *InMemoryStore
}

func (cc *ChatCompletion) Prompt(ctx context.Context, data *PromptRequest, sessionID string) (*ChatMessage, error) {
	// Retrieve conversation history from Redis
	history, err := cc.store.GetChatHistory(sessionID)
	if err != nil {
		return nil, err
	}

	// Prepare the API request messages, including the conversation history
	messages := make([]openai.ChatCompletionMessage, len(history))
	for i, msg := range history {
		messages[i] = openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: msg.Content,
		}
		if msg.Role == System {
			content := fmt.Sprintf("You will be teaching %s, be as concise as possible, answer no more than five sentences.", messages[i].Content)
			messages[i].Content = content
			messages[i].Role = openai.ChatMessageRoleSystem
		}
		if msg.Role == Assistant {
			messages[i].Role = openai.ChatMessageRoleAssistant
		}
	}

	// Add the user's new message to the API request
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: data.Prompt,
	})

	// Call the API with the conversation history
	resp, err := cc.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			Messages:    messages,
			User:        sessionID,
			MaxTokens:   200,
			Temperature: 0.5,
			TopP:        0.5,
		},
	)

	if err != nil {
		return nil, err
	}

	// Save user message to Redis
	userMsg := &ChatMessage{
		Content:   data.Prompt,
		Timestamp: time.Now().Unix(),
		Role:      User,
	}
	err = cc.store.SaveMessage(sessionID, userMsg)

	if err != nil {
		return nil, err
	}

	// Save assistant response to Redis
	assistantMsg := &ChatMessage{
		Content:   resp.Choices[0].Message.Content,
		Timestamp: time.Now().Unix(),
		Role:      Assistant,
	}
	err = cc.store.SaveMessage(sessionID, assistantMsg)

	if err != nil {
		return nil, err
	}

	return assistantMsg, nil
}

func (cc *ChatCompletion) SaveLesson(lessonID string, data *PresetRequest) error {
	return cc.store.SaveLessonPresets(lessonID, data)
}

func (cc *ChatCompletion) CreateSession(lessonID, sessionID, username string) (*SessionCreatedResponse, error) {
	// TODO: Should be transactional??

	err := cc.store.CreateSession(lessonID, sessionID, username)
	if err != nil {
		return nil, err
	}

	preset, err := cc.store.GetLessonPreset(lessonID)
	if err != nil {
		return nil, err
	}

	// TODO: polish system message to make it efficient
	systemMessage := preset

	err = cc.store.SaveMessage(sessionID, &ChatMessage{
		Content:   systemMessage,
		Timestamp: time.Now().Unix(),
		Role:      System,
	})

	response := &SessionCreatedResponse{
		Preset: preset,
	}
	return response, err
}