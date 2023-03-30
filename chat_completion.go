package main

import (
	"context"
	"time"

	"github.com/sashabaranov/go-openai"
)

type ChatMessage struct {
	Content   string `json:"prompt"`
	Timestamp int64  `json:"timestamp"`
	IsUser    bool   `json:"is_user"`
}

type ChatCompletion struct {
	client *openai.Client
	store  *InMemoryStore
}

func (cc *ChatCompletion) Prompt(ctx context.Context, data *PromptRequest, sessionID string) (string, error) {
	resp, err := cc.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: data.Prompt,
				},
			},
			User: sessionID,
		},
	)

	if err != nil {
		return "", err
	}

	msg := &ChatMessage{
		Content:   data.Prompt,
		Timestamp: time.Now().Unix(),
		IsUser:    true,
	}

	err = cc.store.SaveMessage(sessionID, msg)

	if err != nil {
		return "", err
	}

	msg = &ChatMessage{
		Content:   resp.Choices[0].Message.Content,
		Timestamp: time.Now().Unix(),
		IsUser:    false,
	}

	err = cc.store.SaveMessage(sessionID, msg)

	if err != nil {
		return "", err
	}

	return msg.Content, nil

}

func (cc *ChatCompletion) Preset(ctx context.Context, data *PresetRequest) error {
	return nil
}
