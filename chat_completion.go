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
			content := fmt.Sprintf("You will be teaching %s, be as concise as possible, don't give out answers for problems", messages[i].Content)
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
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
			User:     sessionID,
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

// how to send system message on every session start?
// need to push the generated system message to the list, so it can be accessed when retrieving chat history
// I need to push it only once
// so basically I push it when student starts a session.
