package adapters

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	coreAdapters "github.com/shouni/gemini-reviewer-core/pkg/adapters"
)

// LocalGitAdapter は coreAdapters.GitService をimplする

// LocalGitAdapter は、ローカルの 'git' コマンドを subprocess/os/exec 経由で実行するアダプタです。
// coreAdapters.GitService インターフェースを実装します。
type LocalGitAdapter struct {
	LocalPath                string
	SSHKeyPath               string
	BaseBranch               string
	InsecureSkipHostKeyCheck bool
}

// Option はLocalGitAdapterの初期化オプションを設定するための関数です。
type Option func(*LocalGitAdapter)

// WithInsecureSkipHostKeyCheck はSSHホストキーチェックをスキップするオプションを設定します。
func WithInsecureSkipHostKeyCheck(skip bool) Option {
	return func(ga *LocalGitAdapter) {
		ga.InsecureSkipHostKeyCheck = skip
	}
}

// WithBaseBranch はベースブランチを設定するオプションです。
func WithBaseBranch(branch string) Option {
	return func(ga *LocalGitAdapter) {
		ga.BaseBranch = branch
	}
}

// NewLocalGitAdapter は LocalGitAdapter を初期化します。
// 戻り値の型をコアライブラリのインターフェース coreAdapters.GitService に変更
func NewLocalGitAdapter(localPath string, sshKeyPath string, opts ...Option) coreAdapters.GitService {
	adapter := &LocalGitAdapter{
		LocalPath:  localPath,
		SSHKeyPath: sshKeyPath,
		BaseBranch: "main",
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// quotePathForShell は、パスをシェルで安全に使用できるようにシングルクォートで囲みます。
// パス内のシングルクォートもエスケープします。
func quotePathForShell(path string) string {
	// シングルクォートを '\' に置換してエスケープ
	return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
}

// getEnvWithSSH は、現在の環境変数に GIT_SSH_COMMAND を追加したリストを返します。
// GIT_SSH_COMMAND はシェル経由で実行されるため、キーのパスを適切にエスケープします。
func (ga *LocalGitAdapter) getEnvWithSSH() []string {
	env := os.Environ()
	if ga.SSHKeyPath == "" {
		return env
	}

	// コマンドインジェクション脆弱性対策
	safeKeyPath := quotePathForShell(ga.SSHKeyPath)

	// ssh -i '/path/to/key' -F /dev/null ... の形式で構築
	sshCmdParts := []string{"ssh", "-i", safeKeyPath, "-F", "/dev/null"}

	if ga.InsecureSkipHostKeyCheck {
		sshCmdParts = append(sshCmdParts, "-o", "StrictHostKeyChecking=no")
	}

	// スペースで結合してコマンド文字列にする
	sshCmd := strings.Join(sshCmdParts, " ")

	slog.Debug("GIT_SSH_COMMANDを構築", "cmd", sshCmd)

	// GIT_SSH_COMMANDを環境変数に追加
	env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	return env
}

// runGitCommand は、指定されたGitコマンドをアダプタの設定（SSH環境変数など）で実行します。
func (ga *LocalGitAdapter) runGitCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = ga.LocalPath

	// 統一された環境変数設定ロジックを使用
	cmd.Env = ga.getEnvWithSSH()

	slog.Debug("Gitコマンドを実行中", "dir", cmd.Dir, "args", args)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			slog.Error("Gitコマンド実行に失敗しました", "args", args, "stderr", outputStr, "exit", exitErr.ExitCode())
			return "", fmt.Errorf("Gitコマンド実行失敗: %s. 出力:\n%s", exitErr.Error(), outputStr)
		}
		slog.Error("Gitコマンド実行中に予期せぬエラーが発生しました", "args", args, "error", err)
		return "", fmt.Errorf("予期せぬGit実行エラー: %w", err)
	}

	slog.Debug("Gitコマンド成功", "args", args)
	return outputStr, nil
}

// --- coreAdapters.GitService インターフェースの実装 ---

// CloneOrUpdate はリポジトリをクローンするか、既に存在する場合は更新を試みます。
func (ga *LocalGitAdapter) CloneOrUpdate(ctx context.Context, repositoryURL string) error {
	localPath := ga.LocalPath
	info, err := os.Stat(localPath)
	if err == nil {
		// localPath が存在する
		if !info.IsDir() {
			return fmt.Errorf("ローカルパス '%s' はディレクトリではありません。Gitリポジトリをクローンできません。", localPath)
		}
		// localPath がディレクトリの場合、.git ディレクトリの存在を確認
		_, gitDirErr := os.Stat(filepath.Join(localPath, ".git"))
		if os.IsNotExist(gitDirErr) {
			// localPath はディレクトリだが、.git がない -> Gitリポジトリではない
			return fmt.Errorf("ローカルパス '%s' は存在しますが、Gitリポジトリではありません。手動で削除するか、別のパスを指定してください。", localPath)
		} else if gitDirErr != nil {
			// .git ディレクトリの確認中にエラー
			return fmt.Errorf("ローカルパス '%s' 内の .git ディレクトリの確認に失敗しました: %w", localPath, gitDirErr)
		}
		// .git ディレクトリが存在する -> 既存リポジトリとして扱う
		slog.Info("既存リポジトリをオープンしました。後続の Fetch に更新を委ねます。", "path", localPath)
		return nil
	} else if os.IsNotExist(err) {
		// localPath が存在しない場合はクローン
		slog.Info("リポジトリが存在しないため、クローンします。", "url", repositoryURL, "path", localPath, "branch", ga.BaseBranch)

		parentDir := filepath.Dir(localPath)
		repoDir := filepath.Base(localPath)

		// 親ディレクトリが存在しない場合は作成
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("親ディレクトリの作成に失敗しました: %w", err)
			}
		}

		// クローン実行
		cloneArgs := []string{"clone", repositoryURL, repoDir}

		cmd := exec.CommandContext(ctx, "git", cloneArgs...)
		cmd.Dir = parentDir

		// SSH認証環境変数を引き継ぐ
		cmd.Env = ga.getEnvWithSSH()

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("リポジトリのクローンに失敗しました: %s\n%s", err.Error(), string(output))
		}
		slog.Info("リポジトリのクローンに成功しました。", "path", localPath)
		return nil
	} else {
		// その他の os.Stat エラー
		return fmt.Errorf("ローカルパス '%s' の確認に失敗しました: %w", localPath, err)
	}
}

// Fetch はリモートから最新の変更を取得します。
func (ga *LocalGitAdapter) Fetch(ctx context.Context) error {
	_, err := ga.runGitCommand(ctx, "fetch", "origin", "--prune")
	if err != nil {
		return fmt.Errorf("リモートからのフェッチに失敗しました: %w", err)
	}
	return nil
}

// GetCodeDiff は指定された2つのブランチ間の純粋な差分を、ローカルの 'git diff' コマンドで取得します。
func (ga *LocalGitAdapter) GetCodeDiff(ctx context.Context, baseBranch, featureBranch string) (string, error) {
	baseRef := fmt.Sprintf("origin/%s", baseBranch)
	featureRef := fmt.Sprintf("origin/%s", featureBranch)

	// 1. 存在チェック
	_, err := ga.runGitCommand(ctx, "rev-parse", "--verify", baseRef)
	if err != nil {
		return "", fmt.Errorf("ベースブランチ '%s' の参照解決に失敗しました: %w", baseRef, err)
	}
	_, err = ga.runGitCommand(ctx, "rev-parse", "--verify", featureRef)
	if err != nil {
		return "", fmt.Errorf("フィーチャーブランチ '%s' の参照解決に失敗しました: %w", featureRef, err)
	}

	// 2. 3点比較 Diff の実行 (git diff base...feature)
	diffArgs := []string{
		"diff",
		fmt.Sprintf("%s...%s", baseRef, featureRef), // 3点リーダー構文を使用
		"--unified=10",
	}

	diffOutput, err := ga.runGitCommand(ctx, diffArgs...)
	if err != nil {
		return "", fmt.Errorf("差分計算に失敗しました: %w", err)
	}

	return diffOutput, nil
}

// CheckRemoteBranchExists は指定されたブランチがリモート 'origin' に存在するか確認します。
func (ga *LocalGitAdapter) CheckRemoteBranchExists(ctx context.Context, branch string) (bool, error) {
	if branch == "" {
		return false, fmt.Errorf("リモートブランチの存在確認に失敗しました: ブランチ名が空です")
	}

	ref := fmt.Sprintf("origin/%s", branch)
	_, err := ga.runGitCommand(ctx, "rev-parse", "--verify", ref)

	if err != nil {
		slog.Debug("リモートブランチが存在しない、またはアクセスできませんでした。", "ref", ref, "error", err)
		return false, nil
	}

	return true, nil
}

// Cleanup はクリーンアップを実行します。
func (ga *LocalGitAdapter) Cleanup(ctx context.Context) error {
	slog.Info("クリーンアップ: fetch -> checkout -B -> clean を実行します。", "path", ga.LocalPath)

	remote := "origin"
	baseBranch := ga.BaseBranch
	baseRef := fmt.Sprintf("%s/%s", remote, baseBranch)

	// 1. リモートの最新情報を取得し、リモート追跡ブランチを更新
	if _, err := ga.runGitCommand(ctx, "fetch", remote); err != nil {
		return fmt.Errorf("クリーンアップ中のフェッチに失敗: %w", err)
	}

	// 2. ベースブランチの強制チェックアウトとリセット (checkout -B)
	checkoutArgs := []string{"checkout", "-B", baseBranch, baseRef}
	if _, err := ga.runGitCommand(ctx, checkoutArgs...); err != nil {
		return fmt.Errorf("クリーンアップ中のチェックアウト/リセットに失敗: %w", err)
	}

	// 3. 追跡されていないファイルも完全に削除 (clean -f -d)
	if _, err := ga.runGitCommand(ctx, "clean", "-f", "-d"); err != nil {
		return fmt.Errorf("クリーンアップ中のクリーンに失敗: %w", err)
	}

	slog.Info("クリーンアップ: ローカルリポジトリはベースブランチの最新状態でクリーンになりました。", "branch", baseBranch)
	return nil
}
