package builder

import (
	"context"
	"fmt"
	"io"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"

	"git-gemini-cli/internal/adapters"
	"git-gemini-cli/internal/app"
	"git-gemini-cli/internal/config"
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

	// 4. Slack Adapter
	slack, err := adapters.NewSlackAdapter(httpClient, cfg.SlackWebhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Slack adapter: %w", err)
	}

	appCtx := &app.Container{
		Config:     cfg,
		RemoteIO:   rio,
		HTTPClient: httpClient,
		Notifier:   slack,
	}

	// 5. Pipeline (Core Logic)
	reviewPipeline, err := buildPipeline(ctx, appCtx.Config, appCtx.RemoteIO, appCtx.Notifier)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize review pipeline: %w", err)
	}
	appCtx.Pipeline = reviewPipeline

	return appCtx, nil
}

// buildRemoteIO は、 I/O コンポーネントを初期化します。
func buildRemoteIO(ctx context.Context) (*app.RemoteIO, error) {
	factory, err := gcsfactory.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS factory: %w", err)
	}
	w, err := factory.OutputWriter()
	if err != nil {
		return nil, fmt.Errorf("failed to create output writer: %w", err)
	}
	s, err := factory.URLSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to create URL signer: %w", err)
	}
	return &app.RemoteIO{
		Factory: factory,
		Writer:  w,
		Signer:  s,
	}, nil
}
