package main

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis"
)

type InMemoryStore struct {
	rClient *redis.Client
}

func (s *InMemoryStore) CreateSession(sessionID string, username string) error {
	sessionTimeout := 24 * time.Hour
	pipe := s.rClient.Pipeline()
	pipe.HSet(sessionID, "username", username)
	pipe.Expire(sessionID, sessionTimeout)
	_, err := pipe.Exec()
	return err
}

func (s *InMemoryStore) GetUserSession(sessionID string) (string, error) {
	val, err := s.rClient.Get(sessionID).Result()

	if err != nil {
		return "", err
	}
	return val, nil
}

func (s *InMemoryStore) SaveMessage(sessionID string, msg *ChatMessage) error {
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	err = s.rClient.RPush(sessionID+":messages", msgData).Err()
	if err != nil {
		return err
	}

	sessionTimeout := 24 * time.Hour
	return s.rClient.Expire(sessionID+":message", sessionTimeout).Err()
}

func (s *InMemoryStore) GetChatHistory(sessionID string) ([]ChatMessage, error) {
	msgList, err := s.rClient.LRange(sessionID+":message", 0, -1).Result()
	if err != nil {
		return nil, err
	}
	messages := make([]ChatMessage, 0, len(msgList))
	for _, messageData := range msgList {
		var message ChatMessage
		err := json.Unmarshal([]byte(messageData), &message)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}
