package pipeline

import (
	"context"
	"errors"
	"testing"

	"git-gemini-cli/internal/domain"
)

// --- Mock 定義 ---

// MockReviewRunner は domain.ReviewRunner のモックです。
type MockReviewRunner struct {
	RunFunc func(ctx context.Context, req domain.ReviewRequest) domain.ReviewProcessOutcome
}

func (m *MockReviewRunner) Run(ctx context.Context, req domain.ReviewRequest) domain.ReviewProcessOutcome {
	return m.RunFunc(ctx, req)
}

// MockPublishRunner は domain.PublishRunner のモックです。
type MockPublishRunner struct {
	// 戻り値を domain.ReviewResult に修正
	RunFunc func(ctx context.Context, req domain.ReviewRequest, outcome domain.ReviewProcessOutcome) (domain.ReviewResult, error)
}

func (m *MockPublishRunner) Run(ctx context.Context, req domain.ReviewRequest, outcome domain.ReviewProcessOutcome) (domain.ReviewResult, error) {
	return m.RunFunc(ctx, req, outcome)
}

// --- テスト本体 ---

func TestReviewPipeline_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := domain.ReviewRequest{
		RepoURL:   "https://github.com/example/repo",
		GCSBucket: "test-bucket",
		GCSPath:   "reports/test.md",
	}

	t.Run("正常系: レビューから公開まで成功すること", func(t *testing.T) {
		t.Parallel()

		// Outcome は最新の構造体に合わせる (StartTimeが必要な場合があるため設定)
		outcome := domain.ReviewProcessOutcome{
			StepName: "Completed",
		}

		// Result は domain.ReviewResult 型を使用
		publishRes := domain.ReviewResult{
			Status: domain.ReviewStatusSuccess,
			GCSURI: req.GCSURI(),
		}

		mockReviewer := &MockReviewRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest) domain.ReviewProcessOutcome {
				return outcome
			},
		}
		mockPublisher := &MockPublishRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest, o domain.ReviewProcessOutcome) (domain.ReviewResult, error) {
				return publishRes, nil
			},
		}

		p := NewReviewPipeline(mockReviewer, mockPublisher)
		err := p.Execute(ctx, req)

		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}
	})

	t.Run("異常系: パブリッシュが失敗した場合はエラーを返す", func(t *testing.T) {
		t.Parallel()

		errPublish := errors.New("publish failed")
		mockReviewer := &MockReviewRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest) domain.ReviewProcessOutcome {
				return domain.ReviewProcessOutcome{}
			},
		}
		mockPublisher := &MockPublishRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest, o domain.ReviewProcessOutcome) (domain.ReviewResult, error) {
				return domain.ReviewResult{}, errPublish
			},
		}

		p := NewReviewPipeline(mockReviewer, mockPublisher)
		err := p.Execute(ctx, req)

		if !errors.Is(err, errPublish) {
			t.Errorf("expected error %v, got %v", errPublish, err)
		}
	})
}
