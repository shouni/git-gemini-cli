package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/shouni/gemini-reviewer-core/ports"

	"git-gemini-cli/internal/domain"
)

// --- Mock 定義 ---

type mockGitService struct {
	// インターフェースを埋め込むことで、未実装のメソッドがあってもコンパイルを通るようにします
	// ただし、テスト中に未実装メソッドが呼ばれるとパニックになるため注意が必要です
	ports.GitService
	cloneFunc          func(ctx context.Context, repoURL string) error
	cleanupFunc        func(ctx context.Context) error
	checkRefExistsFunc func(ctx context.Context, ref string) (bool, error)
	diffFunc           func(ctx context.Context, b, h string) (string, error)
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

// GitFactory のモック
type mockGitFactory struct {
	service ports.GitService
}

func (f *mockGitFactory) Create(url, base string) ports.GitService {
	return f.service
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

type mockPromptGen struct {
	genReviewFunc func(mode, diff string) (string, error)
	genSkipFunc   func(req domain.ReviewRequest) (string, error)
	reportFunc    func(ctx context.Context, params domain.ErrorReportParams) (string, error)
}

func (m *mockPromptGen) GenerateReview(mode, diff string) (string, error) {
	return m.genReviewFunc(mode, diff)
}
func (m *mockPromptGen) GenerateSkipReport(req domain.ReviewRequest) (string, error) {
	return m.genSkipFunc(req)
}
func (m *mockPromptGen) GenerateErrorReport(ctx context.Context, p domain.ErrorReportParams) (string, error) {
	return "error report", nil
}

// --- テスト本体 ---

func TestReviewRunner_Run(t *testing.T) {
	t.Parallel() // 親テストの並行実行

	ctx := context.Background()
	req := domain.ReviewRequest{
		RepoURL:       "https://github.com/owner/repo",
		BaseBranch:    "main",
		FeatureBranch: "feature",
		Mode:          "default",
		ModelName:     "gemini-1.5-pro",
	}

	tests := []struct {
		name      string
		setupMock func(g *mockGitService, ai *mockCodeReviewAI, pg *mockPromptGen)
		wantSkip  bool
		wantErr   bool
	}{
		{
			name: "正常系: レビューが正常に完了する",
			setupMock: func(g *mockGitService, ai *mockCodeReviewAI, pg *mockPromptGen) {
				g.diffFunc = func(ctx context.Context, b, f string) (string, error) { return "some diff", nil }
				pg.genReviewFunc = func(mode, diff string) (string, error) { return "prompt", nil }
				ai.reviewFunc = func(ctx context.Context, mdl, prpt string) (string, error) {
					return "LGTM!", nil
				}
			},
			wantSkip: false,
			wantErr:  false,
		},
		{
			name: "準正常系: 差分がない場合は IsSkipped が true になる",
			setupMock: func(g *mockGitService, ai *mockCodeReviewAI, pg *mockPromptGen) {
				g.diffFunc = func(ctx context.Context, b, f string) (string, error) { return "", nil }
				pg.genSkipFunc = func(r domain.ReviewRequest) (string, error) { return "skipped", nil }
			},
			wantSkip: true,
			wantErr:  false,
		},
		{
			name: "異常系: Gitクローンに失敗した場合はエラーを保持する",
			setupMock: func(g *mockGitService, ai *mockCodeReviewAI, pg *mockPromptGen) {
				g.cloneFunc = func(ctx context.Context, url string) error { return errors.New("clone failed") }
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt // ループ変数のキャプチャ（Go 1.22 未満の場合に必要）
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // サブテストの並行実行

			mGit := &mockGitService{}
			mAI := &mockCodeReviewAI{}
			mPG := &mockPromptGen{}

			// 初期設定: 特定のテストで上書きされない場合のデフォルト挙動
			mGit.cleanupFunc = func(ctx context.Context) error { return nil }
			mGit.checkRefExistsFunc = func(ctx context.Context, ref string) (bool, error) { return true, nil }

			tt.setupMock(mGit, mAI, mPG)

			factory := &mockGitFactory{service: mGit}
			runner := NewReviewRunner(factory, mAI, mPG)

			outcome := runner.Run(ctx, req)

			if (outcome.Error != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", outcome.Error, tt.wantErr)
			}
			if outcome.IsSkipped != tt.wantSkip {
				t.Errorf("Run() IsSkipped = %v, want %v", outcome.IsSkipped, tt.wantSkip)
			}
		})
	}
}
