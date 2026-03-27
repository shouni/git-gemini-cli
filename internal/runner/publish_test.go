package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shouni/gemini-reviewer-core/ports"

	"git-gemini-cli/internal/domain"
)

// --- Mock 定義 ---

type mockPublisher struct {
	publishFunc func(ctx context.Context, uri string, data ports.ReviewData) error
}

func (m *mockPublisher) Publish(ctx context.Context, uri string, data ports.ReviewData) error {
	return m.publishFunc(ctx, uri, data)
}

type mockURLSigner struct {
	signFunc func(ctx context.Context, uri, method string, exp time.Duration) (string, error)
}

func (m *mockURLSigner) GenerateSignedURL(ctx context.Context, uri, method string, exp time.Duration) (string, error) {
	return m.signFunc(ctx, uri, method, exp)
}

type mockNotifier struct {
	notifyFunc func(ctx context.Context, publicURL, storageURI string, req domain.ReviewRequest) error
}

func (m *mockNotifier) Notify(ctx context.Context, publicURL, storageURI string, req domain.ReviewRequest) error {
	return m.notifyFunc(ctx, publicURL, storageURI, req)
}

// --- テスト本体 ---

func TestPublishRunner_Run(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := domain.ReviewRequest{
		RepoURL:   "https://github.com/example/repo",
		GCSBucket: "test-bucket",
		GCSPath:   "reports/test.md",
	}

	tests := []struct {
		name      string
		outcome   domain.ReviewProcessOutcome
		setupMock func(p *mockPublisher, s *mockURLSigner, n *mockNotifier, g *mockPromptGen)
		wantErr   bool
	}{
		{
			name: "正常系: レビュー成功時にGCS保存と通知が行われる",
			outcome: domain.ReviewProcessOutcome{
				ReviewMarkdown: "## Review Result",
				StartTime:      time.Now().Add(-1 * time.Minute),
				Error:          nil,
			},
			setupMock: func(p *mockPublisher, s *mockURLSigner, n *mockNotifier, g *mockPromptGen) {
				p.publishFunc = func(ctx context.Context, uri string, data ports.ReviewData) error {
					return nil
				}
				s.signFunc = func(ctx context.Context, uri, method string, exp time.Duration) (string, error) {
					return "https://signed.url", nil
				}
				n.notifyFunc = func(ctx context.Context, pURL, sURI string, r domain.ReviewRequest) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "異常系: レビュー自体が失敗していてもエラーレポートをパブリッシュする",
			outcome: domain.ReviewProcessOutcome{
				StepName:  "Reviewing",
				Error:     errors.New("ai error"),
				StartTime: time.Now().Add(-1 * time.Minute),
			},
			setupMock: func(p *mockPublisher, s *mockURLSigner, n *mockNotifier, g *mockPromptGen) {
				g.reportFunc = func(ctx context.Context, params domain.ErrorReportParams) (string, error) {
					return "# Error Report", nil
				}
				p.publishFunc = func(ctx context.Context, uri string, data ports.ReviewData) error {
					return nil
				}
				s.signFunc = func(ctx context.Context, uri, method string, exp time.Duration) (string, error) {
					return "https://signed.url", nil
				}
				n.notifyFunc = func(ctx context.Context, pURL, sURI string, r domain.ReviewRequest) error {
					return nil
				}
			},
			// outcome.Error が含まれるため、戻り値の error は non-nil (wantErr: true) になる設計
			wantErr: true,
		},
		{
			name: "致命的エラー: GCSへのパブリッシュ自体が失敗した場合",
			outcome: domain.ReviewProcessOutcome{
				ReviewMarkdown: "## Content",
				StartTime:      time.Now(),
			},
			setupMock: func(p *mockPublisher, s *mockURLSigner, n *mockNotifier, g *mockPromptGen) {
				p.publishFunc = func(ctx context.Context, uri string, data ports.ReviewData) error {
					return errors.New("gcs connection error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mPub := &mockPublisher{}
			mSig := &mockURLSigner{}
			mNot := &mockNotifier{}
			mGen := &mockPromptGen{}
			tt.setupMock(mPub, mSig, mNot, mGen)

			runner := NewPublishRunner(mPub, mSig, mNot, mGen)
			result, err := runner.Run(ctx, req, tt.outcome)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 成功時の基本的な型チェック
			if !tt.wantErr && result.Status != domain.ReviewStatusSuccess {
				t.Errorf("Expected success status, got %v", result.Status)
			}
		})
	}
}
