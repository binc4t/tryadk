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
	"google.golang.org/genai"
)

type attackArgs struct {
	Target string `json:"target"`
}

type attackResults struct {
	Result string `json:"result"`
}

type moveArgs struct {
	Direction string `json:"direction"`
}

type moveResults struct {
	Result string `json:"result"`
}

func initTools() ([]tool.Tool, error) {
	attackFunc := func(ctx tool.Context, args attackArgs) (attackResults, error) {
		return attackResults{Result: "Attacking " + args.Target}, nil
	}
	attackTool, err := functiontool.New(
		functiontool.Config{
			Name:        "attack",
			Description: "Attack a target",
		},
		attackFunc,
	)
	if err != nil {
		return nil, err
	}

	moveFunc := func(ctx tool.Context, args moveArgs) (moveResults, error) {
		return moveResults{Result: "Moving " + args.Direction}, nil
	}
	moveTool, err := functiontool.New(
		functiontool.Config{
			Name:        "move",
			Description: "Move in a direction",
		},
		moveFunc,
	)
	if err != nil {
		return nil, err
	}

	return []tool.Tool{attackTool, moveTool}, nil
}

func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-3-flash-preview", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	tools, err := initTools()
	if err != nil {
		log.Fatalf("Failed to initialize tools: %v", err)
	}

	instruction := `
	You are an NPC of a game, you must use the tools provided to you to move and attack.

	When you are encountered by a player, you move to the player.
	When you are attacked by a player, you attack the player.
	When you meet a monster, you attack the monster.

	You must always use the tools and output the result of the tools.
	`

	npcAgent, err := llmagent.New(llmagent.Config{
		Name:        "npc_agent",
		Model:       model,
		Description: "NPC of a game, can move and attack.",
		Instruction: instruction,
		Tools:       tools,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	config := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(npcAgent),
	}

	l := full.NewLauncher()
	if err = l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
