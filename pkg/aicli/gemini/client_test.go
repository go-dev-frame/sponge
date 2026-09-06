package gemini

import (
	"context"
	"fmt"
	"testing"
	"time"
)

var apiKey = "xxxxxx"

func TestClient_Send(t *testing.T) {
	client, err := NewClient(apiKey)
	if err != nil {
		t.Log(err)
		return
	}
	defer client.Close()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	replyContent, err := client.Send(ctx, "Who are you?")
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(replyContent)
}

func TestClient_SendStream(t *testing.T) {
	client, err := NewClient(apiKey,
		WithModel(Model25Pro),
		WithEnableContext(),
		WithInitialContextMessages(&ContextMessage{Role: "model", Content: "Hello, I am a gopher."}),
	)
	if err != nil {
		t.Log(err)
		return
	}
	defer client.Close()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	reply := client.SendStream(ctx, "Which model did you use to answer the question?")
	for content := range reply.Content {
		fmt.Print(content)
	}
	if reply.Err != nil {
		t.Log(reply.Err)
		return
	}
}

func TestClient_ListModelNames(t *testing.T) {
	client, err := NewClient(apiKey)
	if err != nil {
		t.Log(err)
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	modelNames, err := client.ListModelNames(ctx)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(modelNames)
}

func TestClient_ListContextsMessages(t *testing.T) {
	client, err := NewClient(apiKey, WithInitialContextMessages([]*ContextMessage{
		{
			Role:    "system",
			Content: "you are a gopher",
		}, {
			Role:    "user",
			Content: "what is goroutine?",
		}, {
			Role:    "",
			Content: "A goroutine is a lightweight, concurrent execution thread in Go.",
		},
	}...))
	if err != nil {
		t.Log(err)
		return
	}

	messages := client.ListContextMessages()
	t.Log(messages)
}
