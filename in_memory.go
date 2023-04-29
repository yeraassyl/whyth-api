package main

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis"
)

type InMemoryStore struct {
	rClient *redis.Client
}

func (s *InMemoryStore) CreateSession(lessonID, sessionID string, username string) error {

	// Getting the expiration time
	expiresUnix, err := s.rClient.HGet(lessonID, "expires").Int64()
	if err != nil {
		return err
	}
	expires := time.Unix(expiresUnix, 0)
	preset, err := s.rClient.HGet(lessonID, "preset").Result()
	if err != nil {
		return err
	}

	pipe := s.rClient.Pipeline()

	// Set username and expiration
	pipe.HSet(sessionID, "username", username)
	pipe.HSet(sessionID, "preset", preset)
	pipe.PExpireAt(sessionID, expires)

	// Add session to the list
	pipe.SAdd(lessonID+":sessions", sessionID)

	_, err = pipe.Exec()
	return nil
}

func (s *InMemoryStore) GetStudentSessionForLesson(lessonID string) ([]string, error) {
	sessionIDs, err := s.rClient.SMembers(lessonID + ":sessions").Result()
	if err != nil {
		return nil, err
	}
	return sessionIDs, nil
}

func (s *InMemoryStore) GetUserSession(sessionID string) (string, error) {
	val, err := s.rClient.HGet(sessionID, "username").Result()

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
	return s.rClient.Expire(sessionID+":messages", sessionTimeout).Err()
}

func (s *InMemoryStore) GetChatHistory(sessionID string) ([]ChatMessage, error) {
	msgList, err := s.rClient.LRange(sessionID+":messages", 0, -1).Result()
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

func (s *InMemoryStore) SaveLessonPresets(lessonID string, presets *PresetRequest) error {
	sessionTimeout := 24 * time.Hour
	expires := time.Now().Add(sessionTimeout)
	pipe := s.rClient.Pipeline()

	pipe.HSet(lessonID, "preset", presets.Preset)
	pipe.HSet(lessonID, "expires", expires.Unix())
	pipe.SAdd(lessonID+":sessions", "temp")
	pipe.SPop(lessonID + ":sessions")
	pipe.PExpireAt(lessonID, expires)
	pipe.PExpireAt(lessonID+":sessions", expires)

	_, err := pipe.Exec()
	return err
}

func (s *InMemoryStore) GetLessonPreset(lessonID string) (string, error) {
	return s.rClient.HGet(lessonID, "preset").Result()
}

func (s *InMemoryStore) GetLessonPreset2(sessionID string) (string, error) {
	return s.rClient.HGet(sessionID, "preset").Result()
}
