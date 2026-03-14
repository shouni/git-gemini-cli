package domain

import (
	"git-gemini-cli/internal/config"
)

type ReviewRequest struct {
	Config         config.Config
	ReviewMarkdown string
	StorageURI     string
}
