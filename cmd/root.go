package cmd

import (
	"context"
	"fmt"
	"git-gemini-cli/internal/config"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/shouni/go-cli-base"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-utils/urlpath"
	"github.com/spf13/cobra"
)

// ReviewConfig は、レビュー実行のパラメータです
var ReviewConfig config.ReviewConfig

const (
	defaultHTTPTimeout = 30 * time.Second
	baseRepoDirName    = "reviewerRepos"
)

// clientKey は context.Context に httpkit.Client を格納・取得するための非公開キー
type clientKey struct{}

// GetHTTPClient は、cmd.Context() から httpkit.ClientInterface を取り出す公開関数です。
func GetHTTPClient(ctx context.Context) (httpkit.ClientInterface, error) {
	if client, ok := ctx.Value(clientKey{}).(httpkit.ClientInterface); ok {
		return client, nil
	}
	return nil, fmt.Errorf("contextからhttpkit.ClientInterfaceを取得できませんでした。")
}

// initAppPreRunE は、アプリケーション固有のPersistentPreRunEです。
func initAppPreRunE(cmd *cobra.Command, args []string) error {

	// ユーザー入力の前後にある余計なスペースを除去
	ReviewConfig.Normalize()

	// slog ハンドラの設定
	logLevel := slog.LevelInfo
	if clibase.Flags.Verbose {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{ // 標準エラー出力にログを出すのが一般的
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// HTTPクライアントの初期化
	httpClient := httpkit.New(defaultHTTPTimeout)

	// RepoURLが指定されている場合のみ、LocalPathの動的生成を試みる
	if ReviewConfig.LocalPath == "" && ReviewConfig.RepoURL != "" {
		ReviewConfig.LocalPath = urlpath.SanitizeURLToUniquePath(ReviewConfig.RepoURL, baseRepoDirName)
		slog.Debug("LocalPathが未指定のため、URLから動的にパスを生成しました。", "generatedPath", ReviewConfig.LocalPath)
	}

	// コマンドのコンテキストに HTTP Client を格納
	ctx := context.WithValue(cmd.Context(), clientKey{}, httpClient)
	cmd.SetContext(ctx)

	return nil
}

// getDefaultSSHKeyPath は、ユーザーのホームディレクトリに基づいてSSH秘密鍵のデフォルトパスを解決します。
func getDefaultSSHKeyPath() string {
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".ssh", "id_rsa")
	}

	return ""
}

// --- フラグ設定ロジック ---
// addAppPersistentFlags は、アプリケーション固有の永続フラグをルートコマンドに追加します。
func addAppPersistentFlags(rootCmd *cobra.Command) {
	defaultSSHKeyPath := getDefaultSSHKeyPath()
	// ReviewConfig.ReviewMode にバインド
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.ReviewMode, "mode", "m", "detail", "レビューモードを指定: 'release' (リリース判定) または 'detail' (詳細レビュー)")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.RepoURL, "repo-url", "u", "", "レビュー対象の Git リポジトリの SSH URL。")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.BaseBranch, "base-branch", "b", "main", "差分比較の基準ブランチ (例: 'main').")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.FeatureBranch, "feature-branch", "f", "", "レビュー対象のフィーチャーブランチ (例: 'feature/my-branch').")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.LocalPath, "local-path", "l", "", "リポジトリをクローンするローカルパス。")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.GeminiModel, "gemini", "g", "gemini-2.5-flash", "レビューに使用する Gemini モデル名 (例: 'gemini-2.5-flash').")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.SSHKeyPath, "ssh-key-path", "k", defaultSSHKeyPath, "Git 認証に使用する SSH 秘密鍵のパス。")
	rootCmd.PersistentFlags().BoolVar(&ReviewConfig.SkipHostKeyCheck, "skip-host-key-check", false, "【🚨 危険な設定】 SSH ホストキーの検証を無効にします。中間者攻撃のリスクを劇的に高めるため、本番環境では絶対に使用しないでください。開発/テスト環境でのみ使用してください。")
	rootCmd.PersistentFlags().BoolVar(&ReviewConfig.UseExternalGitCommand, "use-external-git-command", true, "Go実装の内部アダプターではなく、外部のローカルGitコマンド（git）を使用してリポジトリを操作します。")

	rootCmd.MarkPersistentFlagRequired("repo-url")
	rootCmd.MarkPersistentFlagRequired("feature-branch")
}

// --- エントリポイント ---

// Execute は、clibase.Execute を使用してルートコマンドの構築と実行を委譲します。
func Execute() {
	clibase.Execute(
		"git-gemini-cli",
		addAppPersistentFlags,
		initAppPreRunE,
		genericCmd,
		publishCmd,
	)
}
