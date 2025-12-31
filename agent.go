package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/adk/tool/geminitool"
	"google.golang.org/genai"
)

type attackArgs struct {
	Target string `json:"target"`
}

type attackResults struct {
	Result string `json:"result"`
}

func main() {
	attackFunc := func(ctx tool.Context, args string) (string, error) {
		return "Attacking " + args, nil
	}

	attackTool, err := functiontool.New(
		functiontool.Config{
			Name:        "attack",
			Description: "Attack a target",
		},
		attackFunc,
	)
	if err != nil {
		log.Fatalf("Failed to create attack tool: %v", err)
	}

	moveFunc := func(ctx tool.Context, direction string) (string, error) {
		return "Moving " + direction, nil
	}

	moveTool, err := functiontool.New(
		functiontool.Config{
			Name:        "move",
			Description: "Move in a direction",
		},
		moveFunc,
	)
	if err != nil {
		log.Fatalf("Failed to create move tool: %v", err)
	}

	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-3-flash-preview", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	timeAgent, err := llmagent.New(llmagent.Config{
		Name:        "hello_time_agent",
		Model:       model,
		Description: "NPC of a game, can move and attack.",
		Instruction: "You are an NPC of a game, and you can move and attack.",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
			attackTool,
			moveTool,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	config := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(timeAgent),
	}

	l := full.NewLauncher()
	if err = l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
