package api

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-redis/redis"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sashabaranov/go-openai"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	config, err := Read()
	if err != nil {
		log.Fatal(err)
	}

	client := openai.NewClient(config.ApiKey)

	rClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
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

	errChan := make(chan error)
	go func() {
		if err := s.Start(":" + config.ServerPort); err != nil {
			errChan <- err
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-c:
	case err := <-errChan:
		log.Fatal(err)
	}
	cancel()
}
