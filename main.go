package main

import (
	"github.com/go-redis/redis"
	"github.com/labstack/echo/v4"
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
	s.Use(CheckSessionMiddleware(store))
	s.POST("/login", StartSession(store))
	s.POST("/t_prompt", TeacherPreset())
	s.POST("/prompt", StudentPrompt(service))
	// save the teacher preset
	// separate the logic for teachers and students
	// start session endpoint for students
	// sessions_id using links
	s.Logger.Fatal(s.Start(":1337"))
}
