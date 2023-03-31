package main

import (
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PromptRequest struct {
	Prompt string `json:"prompt"`
}

type PresetRequest struct {
	Preset string `json:"preset"`
}

type StartSessionRequest struct {
	Name     string `json:"name"`
	LessonID string `json:"lesson_id"`
}

func StudentPrompt(cc *ChatCompletion) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := &PromptRequest{}
		err := c.Bind(data)

		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "error unmarshalling the request body")
		}
		sessionID := c.Get("sessionID").(string)

		resp, err := cc.Prompt(c.Request().Context(), data, sessionID)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "can't access the gpt")
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func CreateLesson(cc *ChatCompletion) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := &PresetRequest{}
		err := c.Bind(data)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "error unmarshalling the request body")
		}
		// TODO: Generate lessonID somehow
		lessonID := ""
		err = cc.SaveLesson(lessonID, data)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "can't create s lesson")
		}
		// TODO: Generate link somehow
		link := lessonID

		return c.JSON(http.StatusOK, link)
	}
}

func StartSession(store *InMemoryStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := &StartSessionRequest{}
		err := c.Bind(data)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "error unmarshalling the request body")
		}

		sessionID := uuid.New().String()
		cookie := http.Cookie{Name: "session_id", Value: sessionID, HttpOnly: true, Expires: time.Now().Add(time.Hour * 24)}
		c.SetCookie(&cookie)

		err = store.CreateSession(data.LessonID, sessionID, data.Name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		// TODO: set the url
		return c.Redirect(http.StatusSeeOther, "/")
	}
}

func CheckSessionMiddleware(store *InMemoryStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session, err := c.Cookie("sessionID")
			if err == nil && session.Value != "" {
				return c.Redirect(http.StatusSeeOther, "/login")
			}

			username, err := store.GetUserSession(session.Value)
			if err == redis.Nil {
				return c.Redirect(http.StatusTemporaryRedirect, "/login")
			} else if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			}
			c.Set("username", username)
			c.Set("sessionID", session.Value)

			return next(c)
		}
	}
}

// sessionID, name.
// sessionID chat_history
// how to store the chat_history?
// echo-contrib/session?
