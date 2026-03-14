package domain

import (
	"errors"
	"git-gemini-cli/internal/config"
)

type ReviewRequest struct {
	Config         config.Config
	ReviewMarkdown string
	StorageURI     string
}

// ErrSkipReview は、レビュー対象の差分が存在しないためにパイプラインがスキップされたことを示すエラーです。
var ErrSkipReview = errors.New("差分が見つからなかったためレビューをスキップしました")
