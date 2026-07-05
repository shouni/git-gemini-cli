package cmd

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-remote-io/remoteio"
	"github.com/shouni/go-utils/giturl"
	"github.com/spf13/cobra"

	"github.com/shouni/git-gemini-cli/internal/builder"
	"github.com/shouni/git-gemini-cli/internal/config"
)

// publishCmd は 'publish' サブコマンドを定義します。
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "AIレビュー結果をHTMLに変換し、指定されたGCS/S3 URIに保存します。",
	Long:  `このコマンドは、AIレビュー結果をスタイル付きHTMLに変換した後、go-remote-io を利用してURIスキームに応じたクラウドストレージ（gs:// または s3://）にアップロードします。`,
	Args:  cobra.NoArgs,
	RunE:  publishCommand,
}

func init() {
	publishCmd.Flags().StringVar(&opts.GCSBucket, "bucket", "", "保存先のGCSバケット名")
	if err := publishCmd.MarkFlagRequired("bucket"); err != nil {
		panic(err)
	}
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// publishCommand は、AIによるレビュー結果を生成し、指定されたURIのクラウドストレージに
// 公開（アップロード）と通知を行う publish コマンドの実行ロジックです。
func publishCommand(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	appCtx, err := builder.BuildContainer(ctx, &opts)
	if err != nil {
		return fmt.Errorf("アプリケーションコンテキストの構築に失敗しました: %w", err)
	}
	defer func() {
		slog.Info("♻️ アプリケーションコンテキストをクローズ中...")
		appCtx.Close()
	}()

	// 保存先 URI の決定
	now := time.Now().Format("20060102_150405")
	repoID := giturl.GenerateGCSKeyName(opts.RepoURL)
	safeBranchName := strings.ReplaceAll(opts.FeatureBranch, "/", "-")
	path := fmt.Sprintf("reviews/%s/%s_%s.html", repoID, now, safeBranchName)
	storageURI := remoteio.BuildGCSURI(opts.GCSBucket, path)
	publicURL, err := appCtx.RemoteIO.Signer.GenerateSignedURL(ctx, storageURI, "GET", config.SignedURLExpiration)
	if err != nil {
		slog.ErrorContext(ctx, "署名付きURLの生成失敗", "error", err)
		return fmt.Errorf("署名付きURLの生成に失敗しました: %w", err)
	}

	// 1. domain.ReviewRequest 定義に合わせてフィールドを埋める
	req := ports.ReviewRequest{
		RepoURL:       opts.RepoURL,
		BaseBranch:    opts.BaseBranch,
		FeatureBranch: opts.FeatureBranch,
		Mode:          opts.ReviewMode,
		ModelName:     opts.GeminiModel,
		StorageURI:    storageURI,
		PublicURL:     publicURL,
	}

	// 2. パイプラインの実行（Execute は error を返します）
	if err := appCtx.Pipeline.Execute(ctx, req); err != nil {
		return fmt.Errorf("レビューおよび公開パイプラインの実行に失敗しました: %w", err)
	}

	// 3. 完了ログを出力（req.GCSURI() メソッドを使用してフルパスを表示）
	slog.Info("処理完了", "uri", req.StorageURI)

	return nil
}
