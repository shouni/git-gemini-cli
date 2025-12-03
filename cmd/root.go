package cmd

import (
	"context"
	"fmt"
	"git-gemini-reviewer-go/internal/builder"
	"git-gemini-reviewer-go/internal/config"
	"log/slog"
	"os"

	"github.com/shouni/go-cli-base"
	"github.com/shouni/go-utils/urlpath"
	"github.com/spf13/cobra"
)

// ReviewConfig は、レビュー実行のパラメータです
var ReviewConfig config.ReviewConfig

// initAppPreRunE は、アプリケーション固有のPersistentPreRunEです。
func initAppPreRunE(cmd *cobra.Command, args []string) error {

	// 1. slog ハンドラの設定
	logLevel := slog.LevelInfo
	if clibase.Flags.Verbose {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
	slog.Info("アプリケーション設定初期化完了", slog.String("mode", ReviewConfig.ReviewMode))

	return nil
}

// --- フラグ設定ロジック ---

// addAppPersistentFlags は、アプリケーション固有の永続フラグをルートコマンドに追加します。
func addAppPersistentFlags(rootCmd *cobra.Command) {
	// ReviewConfig.ReviewMode にバインド
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.ReviewMode, "mode", "m", "detail", "レビューモードを指定: 'release' (リリース判定) または 'detail' (詳細レビュー)")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.RepoURL, "repo-url", "u", "", "レビュー対象の Git リポジトリの SSH URL。")
	rootCmd.MarkPersistentFlagRequired("repo-url")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.BaseBranch, "base-branch", "b", "main", "差分比較の基準ブランチ (例: 'main').")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.FeatureBranch, "feature-branch", "f", "", "レビュー対象のフィーチャーブランチ (例: 'feature/my-branch').")
	rootCmd.MarkPersistentFlagRequired("feature-branch")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.LocalPath, "local-path", "l", "", "リポジトリをクローンするローカルパス。")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.GeminiModel, "gemini", "g", "gemini-2.5-flash", "レビューに使用する Gemini モデル名 (例: 'gemini-2.5-flash').")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.SSHKeyPath, "ssh-key-path", "k", "~/.ssh/id_rsa", "Git 認証に使用する SSH 秘密鍵のパス。")
	rootCmd.PersistentFlags().BoolVar(&ReviewConfig.SkipHostKeyCheck, "skip-host-key-check", false, "【🚨 危険な設定】 SSH ホストキーの検証を無効にします。中間者攻撃のリスクを劇的に高めるため、本番環境では絶対に使用しないでください。開発/テスト環境でのみ使用してください。")
}

// --- エントリポイント ---

// Execute は、clibase.Execute を使用してルートコマンドの構築と実行を委譲します。
func Execute() {
	clibase.Execute(
		"git-gemini-reviewer-go",
		addAppPersistentFlags,
		initAppPreRunE,
		genericCmd,
		publishCmd,
	)
}

// executeReviewPipeline は、すべての依存関係を構築し、レビューパイプラインを実行します。
// 実行結果の文字列とエラーを返します。
func executeReviewPipeline(
	ctx context.Context,
	cfg config.ReviewConfig,
) (string, error) {
	const baseRepoDirName = "reviewerRepos"

	// LocalPathが指定されていない場合、RepoURLから動的に生成しcfgを更新します。
	if cfg.LocalPath == "" {
		cfg.LocalPath = urlpath.SanitizeURLToUniquePath(cfg.RepoURL, baseRepoDirName)
		slog.Debug("LocalPathが未指定のため、URLから動的にパスを生成しました。", "generatedPath", cfg.LocalPath)
	}

	// cfg.UseInternalGitAdapter = true は、Git操作に内部で実装されたアダプタを使用することを示します。
	// これは、os/exec を利用した外部コマンド実行を抽象化したものであり、
	// README.mdで説明されている「外部コマンド実行」の原則に沿っています。
	// この設定により、コアライブラリ側でGitコマンドの実行方法を抽象化し、
	// 将来的な拡張性やテスト容易性を向上させています。
	cfg.UseInternalGitAdapter = true
	reviewRunner, err := builder.BuildReviewRunner(ctx, cfg)
	if err != nil {
		// BuildReviewRunner が内部でアダプタやビルダーの構築エラーをラップして返す
		return "", fmt.Errorf("レビュー実行器の構築に失敗しました: %w", err)
	}

	slog.Info("レビューパイプラインを開始します。")

	reviewResult, err := reviewRunner.Run(ctx, cfg)
	if err != nil {
		return "", err
	}

	if reviewResult == "" {
		slog.Info("Diff がないためレビューをスキップしました。")
		return "", nil
	}

	return reviewResult, nil
}
