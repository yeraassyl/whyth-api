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

// need an identifier for each lesson
// need to connect each student session to this lesson identifier
// also need to save presets on the lesson identifier
// teacher should also have a session, where he can edit? maybe.
// so basically he needs an endpoint saved to his session and lesson.
// but maybe it is easier to do it with registration? I mean if it will be a session, then he can't access the lessons he did.
// is there a use case where he can do it without persistent registration?
// so imagine he just created a user session and presets some params like the subject. The system message will be modified
// on the backend side.
// for the first version we may make the lesson immutable, once created you can't edit it.
// and the teacher won't be able to change anything he will just have a link which he can share.
// this way, there will be a lot of people trying to make lessons themselves and it may cause a high load on the server.
// Unless I make the announcement public, there will be just a few groups of people on alpha-testing.
// ya ne hochu privyazyvat' eto imenno k shkole ili univeru, hotya v etom sluchae project budet bolee seriyezniy i skoree
// vsego poluchit horoshee financirovanie.
// esli sdelat' etu platformu kak open to everyone, u menya budet super horoshaya auditorya. vozmozhno ya poteryau den'gi na hostinge,
// no eto togo stoit
// hochetsya sdelat' platformu maksimalno ez to use
// chem proshe tem luchwe
// i ya budu ego razvivat' tem chto budu sledit' za AI dvyzheniem i budu pridumyvat' reshenie dlya drugih problem v ED.
// mb nauchit' ego na kazakhskom bazarit', ispolzovat' neskolko raznye modeli.
// i sdelat' knopku donata, na podderzhku servera.
// nuzhno byt' transparent v tom chto ya ispolzuu openAI? dumau da.
// ya hochu vse zhe kak to zaforcit' prodvizhenie AI obuchenya v shkolah, i mb budet drugaya versia dlya nih?
// koroche v itoge nado mne zamutit' prostuyu vesh, sdelat' lesson i pryvyazyvat' chelov na lesson.
// so it's almost done when I'll finish the teachers endpoints.
// I need to test everything tho
