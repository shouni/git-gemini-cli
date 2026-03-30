package builder

import (
	"context"
	"fmt"
	"io"

	"git-gemini-cli/internal/adapters"
	"git-gemini-cli/internal/app"
	"git-gemini-cli/internal/config"

	"github.com/shouni/go-http-kit/httpkit"
)

// BuildContainer は外部サービスとの接続を確立し、依存関係を組み立てた app.Container を返します。
func BuildContainer(ctx context.Context, cfg *config.Config) (container *app.Container, err error) {
	var resources []io.Closer
	defer func() {
		if err != nil {
			for _, r := range resources {
				if r != nil {
					_ = r.Close()
				}
			}
		}
	}()

	// 1. HttpClient (全アダプターの基盤)
	httpClient := httpkit.New(config.DefaultHTTPTimeout)

	// 2. I/O Infrastructure (マルチクラウクラウド対応)
	rio, err := buildRemoteIO(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize IO components: %w", err)
	}
	resources = append(resources, rio)

	// 3. Prompt Adapter の構築
	promptGen, err := adapters.NewPromptAdapter()
	if err != nil {
		return nil, fmt.Errorf("PromptAdapter の構築に失敗しました: %w", err)
	}

	// 4. Slack Adapter
	slack, err := adapters.NewSlackAdapter(httpClient, cfg.SlackWebhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Slack adapter: %w", err)
	}

	appCtx := &app.Container{
		Config:    cfg,
		RemoteIO:  rio,
		PromptGen: promptGen,
		Notifier:  slack,
	}

	// 5. Pipeline (Core Logic)
	pipeline, err := buildPipeline(ctx, appCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize review pipeline: %w", err)
	}
	appCtx.Pipeline = pipeline

	return appCtx, nil
}
