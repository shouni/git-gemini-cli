package adapters

import (
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/git"
	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-utils/urlpath"

	"git-gemini-cli/internal/config"
)

// GitFactory は、ports.GitFactory インターフェースを満たす具象型です。
type GitFactory struct {
	sshKeyPath       string
	skipHostKeyCheck bool
	localPath        string
	useExternalGit   bool
}

// コンパイル時に ports.GitFactory インターフェースの実装を保証します
var _ ports.GitFactory = (*GitFactory)(nil)

// NewGitFactory は、GitFactory の新しいインスタンスを生成します。
func NewGitFactory(cfg *config.Config) *GitFactory {
	return &GitFactory{
		sshKeyPath:       cfg.SSHKeyPath,
		skipHostKeyCheck: cfg.SkipHostKeyCheck,
		localPath:        cfg.LocalPath,
		useExternalGit:   cfg.UseExternalGit,
	}
}

// Create は ports.GitFactory インターフェースを満たします。
func (g *GitFactory) Create(repoURL, baseBranch string) ports.GitService {
	if g.localPath != "" {
		g.localPath = g.generateLocalPath(repoURL)
	}
	opts := []git.Option{
		git.WithInsecureSkipHostKeyCheck(g.skipHostKeyCheck),
		git.WithBaseBranch(baseBranch),
	}

	if g.useExternalGit {
		slog.Info("GitService: 外部Gitコマンド利用アダプタ (LocalGitAdapter/os/exec) を使用します。")
		return git.NewGitLocalAdapter(g.localPath, g.sshKeyPath, opts...)
	}

	slog.Info("GitService: コアライブラリのアダプタ (go-git) を使用します。")
	return git.NewGitAdapter(g.localPath, g.sshKeyPath, opts...)
}

// generateLocalPath はリポジトリURLからユニークなローカルパスを生成します。
func (g *GitFactory) generateLocalPath(repoURL string) string {
	const baseRepoDirName = "reviewer-repos"
	return urlpath.SanitizeURLToUniquePath(repoURL, baseRepoDirName)
}
