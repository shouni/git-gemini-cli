package pipeline

import (
	"context"
	"fmt"
	"log/slog"

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
	// 1. レビュー実行（中間結果 Outcome を取得）
	outcome := p.Review(ctx, req)

	// 2. 結果のパブリッシュ（GCS保存、Slack通知、エラーレポート生成など）
	// Outcome 内にエラーが含まれていても、publisher が適切に処理して error を返します。
	result, err := p.publisher.Run(ctx, req, outcome)
	if err != nil {
		return fmt.Errorf("publish runner execution failed for repo %s: %w", req.RepoURL, err)
	}

	// 3. 正常終了のログ記録（ステータスが Success または Skipped の場合）
	slog.InfoContext(ctx, "Review pipeline completed successfully.",
		"repo_url", req.RepoURL,
		"status", result.Status,
		"gcs_uri", result.GCSURI,
	)

	return nil
}

// Review はレビュー処理をします。
func (p *ReviewPipeline) Review(ctx context.Context, req domain.ReviewRequest) domain.ReviewProcessOutcome {
	return p.reviewer.Run(ctx, req)
}
