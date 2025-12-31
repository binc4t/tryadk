package main

import (
	"context"
	"log"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
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

// initTracer 初始化 OpenTelemetry tracer，用于观察模型的输入和输出
func initTracer() (func(), error) {
	f, err := os.Create("out.log")
	if err != nil {
		return nil, err
	}

	// 创建 stdout exporter，将 traces 输出到文件 out.log
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(), // 美化输出格式
		stdouttrace.WithWriter(f),     // 输出到文件 out.log
	)
	if err != nil {
		return nil, err
	}

	// 创建 resource，标识服务信息
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("poet_agent"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// 创建 tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 采样所有 traces
	)

	// 设置全局 tracer provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 返回清理函数
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}, nil
}

func main() {
	ctx := context.Background()

	// 初始化 telemetry
	shutdown, err := initTracer()
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer shutdown()

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
