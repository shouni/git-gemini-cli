package adapters

import (
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/pkg/adapters"
	"github.com/shouni/gemini-reviewer-core/pkg/domain"

	"git-gemini-cli/internal/config"
)

// NewGitService は adapters.GitService のインスタンスを構築する Factory 関数です。
func NewGitService(cfg config.ReviewConfig) domain.GitService {
	if cfg.UseExternalGitCommand {
		slog.Debug("GitService: 外部Gitコマンド利用アダプタ (LocalGitAdapter/os/exec) を使用します。")
		return adapters.NewGitLocalAdapter(
			cfg.LocalPath,
			cfg.SSHKeyPath,
			adapters.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
			adapters.WithBaseBranch(cfg.BaseBranch),
		)
	}

	slog.Debug("GitService: コアライブラリのアダプタ (go-git) を使用します。")
	return adapters.NewGitAdapter(
		cfg.LocalPath,
		cfg.SSHKeyPath,
		adapters.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
		adapters.WithBaseBranch(cfg.BaseBranch),
	)
}
