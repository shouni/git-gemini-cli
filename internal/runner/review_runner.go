package runner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shouni/gemini-reviewer-core/ports"

	"git-gemini-cli/internal/domain"
)

// TemplateData はレビュープロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	DiffContent string
}

// ReviewRunner はコードレビューのビジネスロジックを実行します。
// 必要な依存関係（アダプタ）をフィールドとして保持します。
type ReviewRunner struct {
	gitService    ports.GitService
	codeReviewAI  ports.CodeReviewAI
	promptBuilder domain.PromptBuilder
}

// NewReviewRunner は ReviewRunner の新しいインスタンスを生成します。
// 依存関係はコンストラクタ経由で注入されます。
func NewReviewRunner(
	git ports.GitService,
	codeReviewAI ports.CodeReviewAI,
	pb domain.PromptBuilder,
) *ReviewRunner {
	return &ReviewRunner{
		gitService:    git,
		codeReviewAI:  codeReviewAI,
		promptBuilder: pb,
	}
}

// Run はGit Diffを取得し、Gemini AIでレビューを実行します。
func (r *ReviewRunner) Run(
	ctx context.Context,
	req domain.ReviewRequest,
) (string, error) {

	slog.Info("Gitリポジトリのセットアップと差分取得を開始します。")
	cfg := req.Config
	// Gitリポジトリのクローンまたは更新
	err := r.gitService.CloneOrUpdate(ctx, cfg.RepoURL)
	if err != nil {
		return "", fmt.Errorf("リポジトリのセットアップに失敗しました: %w", err)
	}

	// クリーンアップを遅延実行 (常に実行を保証)
	defer func() {
		if cleanupErr := r.gitService.Cleanup(ctx); cleanupErr != nil {
			slog.Error("Gitリポジトリのクリーンアップに失敗しました。", "error", cleanupErr)
		}
	}()

	// リモートから最新の変更をフェッチ
	if err := r.gitService.Fetch(ctx); err != nil {
		return "", fmt.Errorf("最新の変更のフェッチに失敗しました: %w", err)
	}

	// コード差分を取得
	codeDiff, err := r.gitService.GetCodeDiff(ctx, cfg.BaseBranch, cfg.FeatureBranch)
	if err != nil {
		return "", fmt.Errorf("コード差分の取得に失敗しました: %w", err)
	}

	// レビュー対象の差分なし
	if strings.TrimSpace(codeDiff) == "" {
		return "", domain.ErrSkipReview
	}
	slog.Info("Git差分の取得に成功しました。", "size_bytes", len(codeDiff))

	// プロンプトの生成
	slog.InfoContext(ctx, "AIプロンプトを生成中...", "mode", cfg.ReviewMode)
	data := TemplateData{
		DiffContent: codeDiff,
	}
	finalPrompt, err := r.promptBuilder.Build(cfg.ReviewMode, data)
	if err != nil {
		return "", fmt.Errorf("プロンプトの組み立てに失敗しました: %w", err)
	}

	// AIレビューの実行
	slog.Info("AIによるコードレビューを開始します。", "model", cfg.GeminiModel)

	// Gemini Adapterにレビューを依頼
	reviewResult, err := r.codeReviewAI.ReviewCodeDiff(ctx, cfg.GeminiModel, finalPrompt)
	if err != nil {
		return "", fmt.Errorf("AIレビューの実行に失敗しました: %w", err)
	}

	return reviewResult, nil
}
