package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/shouni/clibase"
	"github.com/spf13/cobra"

	"git-gemini-cli/internal/config"
)

// opts は、レビュー実行のパラメータです
var opts config.Config

// Execute は、clibase.Execute を使用してアプリケーションを構築・実行します。
func Execute() {
	clibase.Execute(clibase.App{
		Name:     "git-gemini-cli",
		AddFlags: addAppPersistentFlags,
		PreRunE:  initAppPreRunE,
		Commands: []*cobra.Command{
			reviewCmd,
			publishCmd,
		},
	})
}

// initAppPreRunE は、コマンド実行前にログ設定やクライアント初期化を行います。
func initAppPreRunE(cmd *cobra.Command, args []string) error {
	opts.FillDefaults(config.LoadConfig())
	opts.Normalize()

	logLevel := slog.LevelInfo
	if clibase.GetConfig().Verbose {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
	return nil
}

// addAppPersistentFlags は、アプリケーション固有の永続フラグをルートコマンドに追加します。
func addAppPersistentFlags(rootCmd *cobra.Command) {
	defaultSSHKeyPath := getDefaultSSHKeyPath()

	rootCmd.PersistentFlags().StringVarP(&opts.ReviewMode, "mode", "m", "detail", "レビューモードを指定: 'release' または 'detail'")
	rootCmd.PersistentFlags().StringVarP(&opts.RepoURL, "repo-url", "u", "", "レビュー対象の Git リポジトリの SSH URL。")
	rootCmd.PersistentFlags().StringVarP(&opts.BaseBranch, "base-branch", "b", "main", "差分比較の基準ブランチ。")
	rootCmd.PersistentFlags().StringVarP(&opts.FeatureBranch, "feature-branch", "f", "", "レビュー対象のフィーチャーブランチ。")
	rootCmd.PersistentFlags().StringVarP(&opts.GeminiModel, "gemini", "g", "gemini-2.5-flash", "使用する Gemini モデル名。")
	rootCmd.PersistentFlags().StringVarP(&opts.SSHKeyPath, "ssh-key-path", "k", defaultSSHKeyPath, "Git 認証に使用する SSH 秘密鍵のパス。")
	rootCmd.PersistentFlags().BoolVar(&opts.SkipHostKeyCheck, "skip-host-key-check", false, "SSH ホストキーの検証を無効にします。")
	rootCmd.PersistentFlags().BoolVar(&opts.UseExternalGit, "use-external-git", true, "外部のローカルGitコマンドを使用してリポジトリを操作します。")

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
