package runner

import (
	"context"
	"errors"
	"testing"

	"git-gemini-cli/internal/config"
	"git-gemini-cli/internal/domain"
)

// --- Mock 定義 ---

type mockGitService struct {
	cloneFunc          func(ctx context.Context, repoURL string) error
	cleanupFunc        func(ctx context.Context) error
	fetchFunc          func(ctx context.Context) error
	checkRefExistsFunc func(ctx context.Context, ref string) (bool, error)
	diffFunc           func(ctx context.Context, base, head string) (string, error)
}

func (m *mockGitService) CloneOrUpdate(ctx context.Context, url string) error {
	if m.cloneFunc == nil {
		return nil
	}
	return m.cloneFunc(ctx, url)
}
func (m *mockGitService) Cleanup(ctx context.Context) error {
	if m.cleanupFunc == nil {
		return nil
	}
	return m.cleanupFunc(ctx)
}
func (m *mockGitService) Fetch(ctx context.Context) error {
	if m.fetchFunc == nil {
		return nil
	}
	return m.fetchFunc(ctx)
}
func (m *mockGitService) CheckRefExists(ctx context.Context, ref string) (bool, error) {
	if m.checkRefExistsFunc == nil {
		return true, nil
	}
	return m.checkRefExistsFunc(ctx, ref)
}
func (m *mockGitService) GetCodeDiff(ctx context.Context, b, h string) (string, error) {
	if m.diffFunc == nil {
		return "", nil
	}
	return m.diffFunc(ctx, b, h)
}

type mockCodeReviewAI struct {
	reviewFunc func(ctx context.Context, model, prompt string) (string, error)
}

func (m *mockCodeReviewAI) ReviewCodeDiff(ctx context.Context, mdl, prpt string) (string, error) {
	if m.reviewFunc == nil {
		return "", nil
	}
	return m.reviewFunc(ctx, mdl, prpt)
}

type mockPromptBuilder struct {
	buildFunc func(mode string, data any) (string, error)
}

func (m *mockPromptBuilder) Build(mode string, data any) (string, error) {
	if m.buildFunc == nil {
		return "", nil
	}
	return m.buildFunc(mode, data)
}

// --- テスト本体 ---

func TestReviewRunner_Run(t *testing.T) {
	ctx := context.Background()
	req := domain.ReviewRequest{
		Config: config.Config{
			RepoURL:       "https://github.com/owner/repo",
			BaseBranch:    "main",
			FeatureBranch: "feature",
			ReviewMode:    "default",
			GeminiModel:   "gemini-1.5-pro",
		},
	}

	tests := []struct {
		name      string
		setupMock func(g *mockGitService, ai *mockCodeReviewAI, pb *mockPromptBuilder)
		want      string
		wantErr   error
	}{
		{
			name: "正常系: レビューが正常に完了する",
			setupMock: func(g *mockGitService, ai *mockCodeReviewAI, pb *mockPromptBuilder) {
				g.cloneFunc = func(ctx context.Context, url string) error { return nil }
				g.cleanupFunc = func(ctx context.Context) error { return nil }
				g.fetchFunc = func(ctx context.Context) error { return nil }
				g.diffFunc = func(ctx context.Context, b, f string) (string, error) { return "some diff", nil }

				pb.buildFunc = func(mode string, data any) (string, error) { return "final prompt", nil }

				ai.reviewFunc = func(ctx context.Context, mdl, prpt string) (string, error) {
					return "LGTM!", nil
				}
			},
			want:    "LGTM!",
			wantErr: nil,
		},
		{
			name: "準正常系: 差分がない場合は ErrSkipReview を返す",
			setupMock: func(g *mockGitService, ai *mockCodeReviewAI, pb *mockPromptBuilder) {
				g.diffFunc = func(ctx context.Context, b, f string) (string, error) { return "", nil }
			},
			want:    "",
			wantErr: domain.ErrSkipReview,
		},
		{
			name: "異常系: Gitクローンに失敗した場合はエラーを返す",
			setupMock: func(g *mockGitService, ai *mockCodeReviewAI, pb *mockPromptBuilder) {
				g.cloneFunc = func(ctx context.Context, url string) error { return errors.New("clone failed") }
			},
			want:    "",
			wantErr: errors.New("リポジトリのセットアップに失敗しました: clone failed"),
		},
		{
			name: "異常系: AIレビュー実行時にエラーが発生した場合はエラーを返す",
			setupMock: func(g *mockGitService, ai *mockCodeReviewAI, pb *mockPromptBuilder) {
				g.diffFunc = func(ctx context.Context, b, f string) (string, error) { return "diff", nil }
				pb.buildFunc = func(mode string, data any) (string, error) { return "prompt", nil }
				ai.reviewFunc = func(ctx context.Context, mdl, prpt string) (string, error) {
					return "", errors.New("api error")
				}
			},
			want:    "",
			wantErr: errors.New("AIレビューの実行に失敗しました: api error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mGit := &mockGitService{}
			mAI := &mockCodeReviewAI{}
			mPB := &mockPromptBuilder{}

			// 初期状態で Cleanup が呼ばれても大丈夫なように設定
			mGit.cleanupFunc = func(ctx context.Context) error { return nil }

			tt.setupMock(mGit, mAI, mPB)

			runner := NewReviewRunner(mGit, mAI, mPB)
			got, err := runner.Run(ctx, req)

			if tt.wantErr != nil {
				if err == nil || (err.Error() != tt.wantErr.Error() && !errors.Is(err, tt.wantErr)) {
					t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Run() unexpected error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Run() got = %v, want %v", got, tt.want)
			}
		})
	}
}
