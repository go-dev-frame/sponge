package deepseek

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mdContent, err := client.Send(ctx, "你是谁？")
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(mdContent)
}

func TestClient_SendStream(t *testing.T) {
	client, err := NewClient(apiKey,
		WithModel(ModelDeepSeekReasoner),
		WithMaxTokens(8192),
		WithTemperature(TemperatureCodeGeneration),
		WithEnableContext(),
		WithInitialRole(genericRoleDescEN),
	)
	if err != nil {
		t.Log(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	answer := client.SendStream(ctx, genericRoleDescZH)
	for content := range answer.Content {
		fmt.Printf(content)
	}
	if answer.Err != nil {
		t.Log(answer.Err)
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

func TestClient_ModifyInitialRole(t *testing.T) {
	client, err := NewClient(apiKey, WithInitialRole(genericRoleDescEN))
	if err != nil {
		t.Log(err)
		return
	}

	client.ModifyInitialRole(genericRoleDescZH) // will recover initial role

	messages := client.ListContextMessages()
	t.Log(messages)
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
			Role:    "assistant",
			Content: "A goroutine is a lightweight, concurrent execution thread in Go.",
		},
	}...))
	if err != nil {
		t.Log(err)
		return
	}

	messages := client.ListContextMessages()
	t.Log(messages)

	client.ModifyInitialRole(genericRoleDescEN) // will recover initial role

	messages = client.ListContextMessages()
	t.Log(messages)
}
