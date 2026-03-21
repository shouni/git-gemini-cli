package pipeline

import (
	"context"
	"errors"
	"testing"

	"git-gemini-cli/internal/domain"
)

// --- Mock 定義 ---

// MockReviewRunner は ReviewRunner のモックです。
type MockReviewRunner struct {
	RunFunc func(ctx context.Context, req domain.ReviewRequest) (string, error)
}

func (m *MockReviewRunner) Run(ctx context.Context, req domain.ReviewRequest) (string, error) {
	return m.RunFunc(ctx, req)
}

type MockPublishRunner struct {
	RunFunc func(ctx context.Context, req domain.ReviewRequest) error
}

func (m *MockPublishRunner) Run(ctx context.Context, req domain.ReviewRequest) error {
	return m.RunFunc(ctx, req)
}

// --- テスト本体 ---

func TestReviewPipeline_Execute(t *testing.T) {
	t.Parallel() // 親テストの並行実行を許可

	ctx := context.Background()
	req := domain.ReviewRequest{
		StorageURI: "gs://bucket/path.md",
	}
	expectedMarkdown := "# Result"

	t.Run("正常系: レビューから公開まで成功すること", func(t *testing.T) {
		t.Parallel() // サブテストの並行実行を許可

		reviewCalled := false
		publishCalled := false

		mockReviewer := &MockReviewRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest) (string, error) {
				reviewCalled = true
				return expectedMarkdown, nil
			},
		}
		mockPublisher := &MockPublishRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest) error {
				publishCalled = true
				return nil
			},
		}

		p := NewReviewPipeline(mockReviewer, mockPublisher)
		err := p.Execute(ctx, req)

		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}
		if !reviewCalled || !publishCalled {
			t.Errorf("runners were not called correctly: review=%v, publish=%v", reviewCalled, publishCalled)
		}
	})

	t.Run("正常系: 差分なし(ErrSkipReview)の場合はエラーにならず公開も呼ばれないこと", func(t *testing.T) {
		t.Parallel()

		reviewCalled := false
		publishCalled := false

		mockReviewer := &MockReviewRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest) (string, error) {
				reviewCalled = true
				return "", domain.ErrSkipReview
			},
		}
		mockPublisher := &MockPublishRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest) error {
				publishCalled = true
				return nil
			},
		}

		p := NewReviewPipeline(mockReviewer, mockPublisher)
		err := p.Execute(ctx, req)

		if err != nil {
			t.Errorf("expected nil error for ErrSkipReview, got %v", err)
		}
		if !reviewCalled {
			t.Error("reviewer should be called")
		}
		if publishCalled {
			t.Error("publisher should not be called when review is skipped")
		}
	})

	t.Run("異常系: レビューが失敗（スキップ以外）した場合はエラーを返す", func(t *testing.T) {
		t.Parallel()

		errReview := errors.New("ai error")
		mockReviewer := &MockReviewRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest) (string, error) {
				return "", errReview
			},
		}
		mockPublisher := &MockPublishRunner{
			RunFunc: func(ctx context.Context, r domain.ReviewRequest) error {
				return nil
			},
		}

		p := NewReviewPipeline(mockReviewer, mockPublisher)
		err := p.Execute(ctx, req)

		if !errors.Is(err, errReview) {
			t.Errorf("expected error %v, got %v", errReview, err)
		}
	})
}
