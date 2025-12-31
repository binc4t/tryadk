package main

import (
	"context"
	"log"
	"os"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

type poemInput struct {
	LineCount int `json:"line_count"`
}

type poemOutput struct {
	Poem string `json:"poem"`
}

func initTools() ([]tool.Tool, error) {
	poemFunc := func(ctx tool.Context, args poemInput) (poemOutput, error) {
		return poemOutput{Poem: strings.Repeat("a line of poem\n", args.LineCount)}, nil
	}
	poemTool, err := functiontool.New(
		functiontool.Config{
			Name:        "poem",
			Description: "Return a poem",
		},
		poemFunc,
	)
	if err != nil {
		return nil, err
	}
	return []tool.Tool{poemTool}, nil
}

func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
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
	When Asked to write a poem, you MUST use tool to write poems.
	`

	npcAgent, err := llmagent.New(llmagent.Config{
		Name:        "poet_agent",
		Model:       model,
		Description: "A poet agent that uses tools to write poems.",
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
