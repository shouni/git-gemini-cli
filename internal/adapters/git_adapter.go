package adapters

import (
	"log/slog"

	coreAdapters "github.com/shouni/gemini-reviewer-core/pkg/adapters"
	"github.com/shouni/gemini-reviewer-core/pkg/ports"

	"git-gemini-cli/internal/config"
)

// NewGitService は adapters.GitService のインスタンスを構築する Factory 関数です。
func NewGitService(cfg *config.Config) ports.GitService {
	if cfg.UseExternalGitCommand {
		slog.Debug("GitService: 外部Gitコマンド利用アダプタ (LocalGitAdapter/os/exec) を使用します。")
		return coreAdapters.NewGitLocalAdapter(
			cfg.LocalPath,
			cfg.SSHKeyPath,
			coreAdapters.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
			coreAdapters.WithBaseBranch(cfg.BaseBranch),
		)
	}

	slog.Debug("GitService: コアライブラリのアダプタ (go-git) を使用します。")
	return coreAdapters.NewGitAdapter(
		cfg.LocalPath,
		cfg.SSHKeyPath,
		coreAdapters.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
		coreAdapters.WithBaseBranch(cfg.BaseBranch),
	)
}
