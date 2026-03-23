package adapters

import (
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/git"
	"github.com/shouni/gemini-reviewer-core/ports"

	"git-gemini-cli/internal/config"
)

// NewGitService は adapters.GitService のインスタンスを構築する Factory 関数です。
func NewGitService(cfg *config.Config) ports.GitService {
	opts := []git.Option{
		git.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
		git.WithBaseBranch(cfg.BaseBranch),
	}

	if cfg.UseExternalGitCommand {
		slog.Debug("GitService: 外部Gitコマンド利用アダプタ (LocalGitAdapter/os/exec) を使用します。")
		return git.NewGitLocalAdapter(cfg.LocalPath, cfg.SSHKeyPath, opts...)
	}

	slog.Debug("GitService: コアライブラリのアダプタ (go-git) を使用します。")
	return git.NewGitAdapter(cfg.LocalPath, cfg.SSHKeyPath, opts...)
}
