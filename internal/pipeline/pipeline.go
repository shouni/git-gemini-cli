package pipeline

import (
	"context"

	"git-gemini-cli/internal/domain"
)

// ReviewPipeline はレビューと公開のランナーを保持し、パイプラインの実行をオーケストレートするサービス構造体です。
type ReviewPipeline struct {
	reviewer  domain.ReviewRunner
	publisher domain.PublishRunner
}

// NewReviewPipeline はレビューと公開のランナーを受け取り、ReviewPipeline を初期化します。
func NewReviewPipeline(reviewer domain.ReviewRunner, publisher domain.PublishRunner) *ReviewPipeline {
	return &ReviewPipeline{
		reviewer:  reviewer,
		publisher: publisher,
	}
}

// Execute はレビューリクエストの全工程（実行から公開まで）をオーケストレートします。
func (p *ReviewPipeline) Execute(ctx context.Context, req domain.ReviewRequest) error {
	result, err := p.Review(ctx, req)
	if err != nil {
		return err
	}
	publishReq := req
	publishReq.ReviewMarkdown = result

	return p.Publish(ctx, publishReq)
}

// Review はレビュー処理をします。
func (p *ReviewPipeline) Review(ctx context.Context, req domain.ReviewRequest) (string, error) {
	return p.reviewer.Run(ctx, req)
}

// Publish は公開処理をします。
func (p *ReviewPipeline) Publish(ctx context.Context, req domain.ReviewRequest) error {
	return p.publisher.Run(ctx, req)
}
