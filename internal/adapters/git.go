package adapters

import (
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/git"
	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-utils/urlpath"

	"git-gemini-cli/internal/config"
)

// GitFactory は、domain.GitFactory インターフェースを満たす具象型です。
type GitFactory struct {
	sshKeyPath            string
	skipHostKeyCheck      bool
	UseExternalGitCommand bool
}

func NewGitFactory(cfg *config.Config) *GitFactory {
	return &GitFactory{
		sshKeyPath:            cfg.SSHKeyPath,
		skipHostKeyCheck:      cfg.SkipHostKeyCheck,
		UseExternalGitCommand: cfg.UseExternalGitCommand,
	}
}

// Create は domain.GitFactory インターフェースを満たします。
func (g *GitFactory) Create(repoURL, baseBranch string) ports.GitService {
	localPath := g.generateLocalPath(repoURL)
	opts := []git.Option{
		git.WithInsecureSkipHostKeyCheck(g.skipHostKeyCheck),
		git.WithBaseBranch(baseBranch),
	}

	if g.UseExternalGitCommand {
		slog.Info("GitService: 外部Gitコマンド利用アダプタ (LocalGitAdapter/os/exec) を使用します。")
		return git.NewGitLocalAdapter(localPath, g.sshKeyPath, opts...)
	}

	slog.Info("GitService: コアライブラリのアダプタ (go-git) を使用します。")
	return git.NewGitAdapter(localPath, g.sshKeyPath, opts...)
}

// generateLocalPath はリポジトリURLからユニークなローカルパスを生成します。
func (g *GitFactory) generateLocalPath(repoURL string) string {
	const baseRepoDirName = "reviewer-repos"
	basePath := urlpath.SanitizeURLToUniquePath(repoURL, baseRepoDirName)

	return basePath
}
