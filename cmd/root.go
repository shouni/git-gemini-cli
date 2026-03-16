package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/shouni/clibase"
	"github.com/shouni/go-utils/urlpath"
	"github.com/spf13/cobra"

	"git-gemini-cli/internal/config"
)

// ReviewConfig は、レビュー実行のパラメータです
var ReviewConfig config.Config

const baseRepoDirName = "reviewerRepos"

// Execute は、clibase.Execute を使用してアプリケーションを構築・実行します。
func Execute() {
	clibase.Execute(clibase.App{
		Name:     "git-gemini-cli",
		AddFlags: addAppPersistentFlags,
		PreRunE:  initAppPreRunE,
		Commands: []*cobra.Command{
			genericCmd,
			publishCmd,
		},
	})
}

// initAppPreRunE は、コマンド実行前にログ設定やクライアント初期化を行います。
func initAppPreRunE(cmd *cobra.Command, args []string) error {
	ReviewConfig.FillDefaults(config.LoadConfig())
	ReviewConfig.Normalize()

	// slog ハンドラの設定 (clibase.GetConfig().Verbose を参照)
	logLevel := slog.LevelInfo
	if clibase.GetConfig().Verbose {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// RepoURLが指定されている場合のみ、LocalPathの動的生成を試みる
	if ReviewConfig.LocalPath == "" && ReviewConfig.RepoURL != "" {
		ReviewConfig.LocalPath = urlpath.SanitizeURLToUniquePath(ReviewConfig.RepoURL, baseRepoDirName)
		slog.Debug("LocalPathが未指定のため、URLから動的にパスを生成しました。", "generatedPath", ReviewConfig.LocalPath)
	}

	return nil
}

// addAppPersistentFlags は、アプリケーション固有の永続フラグをルートコマンドに追加します。
func addAppPersistentFlags(rootCmd *cobra.Command) {
	defaultSSHKeyPath := getDefaultSSHKeyPath()

	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.ReviewMode, "mode", "m", "detail", "レビューモードを指定: 'release' または 'detail'")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.RepoURL, "repo-url", "u", "", "レビュー対象の Git リポジトリの SSH URL。")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.BaseBranch, "base-branch", "b", "main", "差分比較の基準ブランチ。")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.FeatureBranch, "feature-branch", "f", "", "レビュー対象のフィーチャーブランチ。")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.LocalPath, "local-path", "l", "", "リポジトリをクローンするローカルパス。")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.GeminiModel, "gemini", "g", "gemini-2.5-flash", "使用する Gemini モデル名。")
	rootCmd.PersistentFlags().StringVarP(&ReviewConfig.SSHKeyPath, "ssh-key-path", "k", defaultSSHKeyPath, "Git 認証に使用する SSH 秘密鍵のパス。")
	rootCmd.PersistentFlags().BoolVar(&ReviewConfig.SkipHostKeyCheck, "skip-host-key-check", false, "SSH ホストキーの検証を無効にします。")
	rootCmd.PersistentFlags().BoolVar(&ReviewConfig.UseExternalGitCommand, "use-external-git-command", true, "外部のローカルGitコマンドを使用してリポジトリを操作します。")

	_ = rootCmd.MarkPersistentFlagRequired("repo-url")
	_ = rootCmd.MarkPersistentFlagRequired("feature-branch")
}

// getDefaultSSHKeyPath は、ユーザーのホームディレクトリに基づいてSSH秘密鍵のデフォルトパスを解決します。
func getDefaultSSHKeyPath() string {
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".ssh", "id_rsa")
	}
	return ""
}
