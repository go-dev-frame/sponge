package chatgpt

import (
	"context"
	"fmt"
	"testing"
	"time"
)

var apiKey = "sk-xxxxxx"

const (
	genericRoleDescEN = "You are a helpful assistant, able to answer user questions in a clear and friendly manner."
	genericRoleDescZH = "你是一个有帮助的助手，能够以清晰、友好的方式回答用户的问题。"
)

func TestClient_Send(t *testing.T) {
	client, err := NewClient(apiKey)
	if err != nil {
		t.Log(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	replyContent, err := client.Send(ctx, "Who are you?")
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(replyContent)
}

func TestClient_SendStream(t *testing.T) {
	client, err := NewClient(apiKey,
		WithModel(ModelGPT4o),
		WithMaxTokens(8192),
		WithEnableContext(),
		WithTemperature(0.3),
		WithInitialRole(genericRoleDescEN),
	)
	if err != nil {
		t.Log(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reply := client.SendStream(ctx, "Which model did you use to answer the question?")
	for content := range reply.Content {
		fmt.Printf(content)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	modelNames, err := client.ListModelNames(ctx)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(modelNames)
}
