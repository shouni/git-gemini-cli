package pipeline

import (
	"context"
	"errors"

	"git-gemini-cli/internal/domain"
)

// ErrSkipReview は、レビュー対象の差分が存在しないためにパイプラインがスキップされたことを示すエラーです。
var ErrSkipReview = errors.New("差分が見つからなかったためレビューをスキップしました")

// ReviewPipeline はパイプラインの実行に必要な外部依存関係を保持するサービス構造体です。
type ReviewPipeline struct {
	reviewRunner  domain.ReviewRunner
	publishRunner domain.PublishRunner
}

func NewReviewPipeline(r domain.ReviewRunner, p domain.PublishRunner) *ReviewPipeline {
	return &ReviewPipeline{
		reviewRunner:  r,
		publishRunner: p,
	}
}

// Execute はレビューリクエストの全工程（実行から公開まで）をオーケストレートします。
func (p *ReviewPipeline) Execute(ctx context.Context, req domain.ReviewRequest) error {
	result, err := p.reviewRunner.Run(ctx, req)
	if err != nil {
		return err
	}
	if result == "" {
		return ErrSkipReview
	}
	publishReq := req
	publishReq.ReviewMarkdown = result

	return p.publishRunner.Run(ctx, publishReq)
}

// Review はレビュー処理をします。
func (p *ReviewPipeline) Review(ctx context.Context, req domain.ReviewRequest) (string, error) {
	return p.reviewRunner.Run(ctx, req)
}

// Publish は公開処理をします。
func (p *ReviewPipeline) Publish(ctx context.Context, req domain.ReviewRequest) error {
	return p.publishRunner.Run(ctx, req)
}
