package main

import (
	"context"
	"fmt"
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
	// Retrieve conversation history from Redis
	history, err := cc.store.GetChatHistory(sessionID)
	if err != nil {
		return "", err
	}

	// Prepare the API request messages, including the conversation history
	messages := make([]openai.ChatCompletionMessage, len(history))
	for i, msg := range history {
		messages[i] = openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: msg.Content,
		}
		if !msg.IsUser {
			messages[i].Role = openai.ChatMessageRoleAssistant
		}
	}

	// Add the user's new message to the API request
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: data.Prompt,
	})

	fmt.Println(messages)

	// Call the API with the conversation history
	resp, err := cc.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
			User:     sessionID,
		},
	)

	if err != nil {
		return "", err
	}

	// Save user message to Redis
	userMsg := &ChatMessage{
		Content:   data.Prompt,
		Timestamp: time.Now().Unix(),
		IsUser:    true,
	}
	err = cc.store.SaveMessage(sessionID, userMsg)

	if err != nil {
		return "", err
	}

	// Save assistant response to Redis
	assistantMsg := &ChatMessage{
		Content:   resp.Choices[0].Message.Content,
		Timestamp: time.Now().Unix(),
		IsUser:    false,
	}
	err = cc.store.SaveMessage(sessionID, assistantMsg)

	if err != nil {
		return "", err
	}

	return assistantMsg.Content, nil
}

func (cc *ChatCompletion) SaveLesson(lessonID string, data *PresetRequest) error {
	return cc.store.SaveLessonPresets(lessonID, data)
}

// how teacher will access the completion?
// need to save the teachers preset in redis
// how do I know which teacher is for which session?
// should teacher register? how should I limit the teachers and students?
// so basically, do you choose which do you want, teacher or students
// if teacher then you can create a lesson, preset some params and then send a link to your students.
// the students will login using a code, or by link you can say. what does this link will do?

// vot esli prosto est' service kuda mozhno pisat' i otvet poluchat', smysla net delat' takoe potomu chto est' uzhe chatgpt
// est' smysl sdelat' AI assitentom dlya uchitelya i dlya studenta.
// sil'no uslozhnyat' ne nuhzno, nuzhno chtoby byl prostoi access so storony uchitelya i uchenika.
// v0.0.1: vozmozhnost' ispolozvoat' AI dlya studenta v obuchenii nekoi temy.
