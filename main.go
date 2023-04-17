package main

import (
	"github.com/go-redis/redis"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sashabaranov/go-openai"
)

func main() {
	client := openai.NewClient("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")

	rClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	store := &InMemoryStore{rClient: rClient}
	service := &ChatCompletion{client: client, store: store}

	s := echo.New()
	s.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}, latency=${latency_human}\n",
	}))
	api := s.Group("/api")
	{
		api.POST("/session", StartSession(service))
		api.POST("/lesson", CreateLesson(service))
		api.POST("/prompt", StudentPrompt(service), CheckSessionMiddleware(store))
		api.GET("/chat-history", ChatHistory(store), CheckSessionMiddleware(store))
	}

	// save the teacher preset
	// separate the logic for teachers and students
	// start session endpoint for students
	// sessions_id using links
	s.Logger.Fatal(s.Start(":1337"))
}
