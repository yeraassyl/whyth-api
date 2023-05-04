package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Role int

const lessonTopic = "You are an AI language model assisting students in learning the topic: '%s'. Focus your responses and guidance on this specific subject and ensure that your interactions are relevant and directly related to the topic being taught. Be concise, limit you response to 3 or 4 sentences. User can always press continue if he wants to know more"

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

	// Prepare the conversation history for the API request
	messages := func(chatHistory []ChatMessage) []openai.ChatCompletionMessage {
		messages := make([]openai.ChatCompletionMessage, len(chatHistory))
		for i, msg := range chatHistory {
			messages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: msg.Content,
			}
			if msg.Role == System {
				messages[i].Role = openai.ChatMessageRoleSystem
			}
			if msg.Role == Assistant {
				messages[i].Role = openai.ChatMessageRoleAssistant
			}
		}
		return messages
	}(history)

	// Add the user's new message to the API request
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: data.Prompt,
	})

	// Call the API with the conversation history
	resp, err := cc.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       openai.GPT4,
			Messages:    messages,
			User:        sessionID,
			MaxTokens:   150,
			Temperature: 0.3,
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

func (cc *ChatCompletion) SaveLesson(lessonID string, data *LessonCreateRequest) error {
	return cc.store.SaveLessonPresets(lessonID, data.LessonName, data.Presets)
}

func (cc *ChatCompletion) CreateSession(lessonID, sessionID, username string) error {
	err := cc.store.CreateSession(lessonID, sessionID, username)
	if err != nil {
		return err
	}

	lessonName, err := cc.store.GetLessonName(lessonID)

	if err != nil {
		return err
	}

	// Set lesson topic as system message
	err = cc.store.SaveMessage(sessionID, &ChatMessage{
		Content:   fmt.Sprintf(lessonTopic, lessonName),
		Timestamp: time.Now().Unix(),
		Role:      System,
	})

	if err != nil {
		return err
	}

	presets, err := cc.store.GetLessonPresets(lessonID)

	if err != nil {
		return err
	}

	// Set presets as system messages for the new session

	for _, preset := range presets {
		if preset.Checked {
			err = cc.store.SaveMessage(sessionID, &ChatMessage{
				Content:   preset.Value,
				Timestamp: time.Now().Unix(),
				Role:      System,
			})
			if err != nil {
				return err
			}
		}
	}

	return err
}
