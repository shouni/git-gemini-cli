package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shouni/gemini-reviewer-core/ports"

	"git-gemini-cli/internal/config"
	"git-gemini-cli/internal/domain"
)

// --- Mock 定義 ---

type mockPublisher struct {
	publishFunc func(ctx context.Context, uri string, data ports.ReviewData) error
}

func (m *mockPublisher) Publish(ctx context.Context, uri string, data ports.ReviewData) error {
	if m.publishFunc == nil {
		return nil
	}
	return m.publishFunc(ctx, uri, data)
}

type mockURLSigner struct {
	signFunc func(ctx context.Context, uri, method string, exp time.Duration) (string, error)
}

// GenerateSignedURL は引数 exp を time.Duration として扱います
func (m *mockURLSigner) GenerateSignedURL(ctx context.Context, uri, method string, exp time.Duration) (string, error) {
	if m.signFunc == nil {
		return "", nil
	}
	return m.signFunc(ctx, uri, method, exp)
}

type mockNotifier struct {
	notifyFunc func(ctx context.Context, publicURL, storageURI string, req domain.ReviewRequest) error
}

func (m *mockNotifier) Notify(ctx context.Context, publicURL, storageURI string, req domain.ReviewRequest) error {
	if m.notifyFunc == nil {
		return nil
	}
	return m.notifyFunc(ctx, publicURL, storageURI, req)
}

// --- テスト本体 ---

func TestPublisherRunner_Run(t *testing.T) {
	ctx := context.Background()

	// 共通のテストリクエスト
	req := domain.ReviewRequest{
		Config: config.Config{
			RepoURL: "https://github.com/example/repo",
		},
		ReviewMarkdown: "## Review Result\n- Good code!",
		StorageURI:     "gs://bucket/path/to/report.md",
	}

	tests := []struct {
		name      string
		setupMock func(p *mockPublisher, s *mockURLSigner, n *mockNotifier)
		wantErr   bool
	}{
		{
			name: "正常系: GCSへのアップロードと署名付きURLでの通知が成功する",
			setupMock: func(p *mockPublisher, s *mockURLSigner, n *mockNotifier) {
				p.publishFunc = func(ctx context.Context, uri string, data ports.ReviewData) error { return nil }
				s.signFunc = func(ctx context.Context, uri, method string, exp time.Duration) (string, error) {
					// 30分が指定されているか検証可能
					if exp != 30*time.Minute {
						return "", errors.New("invalid expiration")
					}
					return "https://signed-url.com", nil
				}
				n.notifyFunc = func(ctx context.Context, pURL, sURI string, r domain.ReviewRequest) error {
					if pURL != "https://signed-url.com" {
						return errors.New("unexpected public URL")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "準正常系: URL署名に失敗しても通知自体は実行される (Fallback)",
			setupMock: func(p *mockPublisher, s *mockURLSigner, n *mockNotifier) {
				p.publishFunc = func(ctx context.Context, uri string, data ports.ReviewData) error { return nil }
				s.signFunc = func(ctx context.Context, uri, method string, exp time.Duration) (string, error) {
					return "", errors.New("sign error")
				}
				n.notifyFunc = func(ctx context.Context, pURL, sURI string, r domain.ReviewRequest) error {
					// 署名失敗時は StorageURI がそのまま渡される
					if pURL != req.StorageURI {
						return errors.New("should use fallback URI")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "異常系: ストレージへの保存に失敗した場合はエラーを返す",
			setupMock: func(p *mockPublisher, s *mockURLSigner, n *mockNotifier) {
				p.publishFunc = func(ctx context.Context, uri string, data ports.ReviewData) error {
					return errors.New("storage error")
				}
				// Publishが失敗した時、署名や通知は呼ばれないことを前提とする（実装通り）
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mPub := &mockPublisher{}
			mSig := &mockURLSigner{}
			mNot := &mockNotifier{}
			tt.setupMock(mPub, mSig, mNot)

			runner := NewPublishRunner(mPub, mSig, mNot)
			err := runner.Run(ctx, req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_convertS3URIToPublicURL(t *testing.T) {
	tests := []struct {
		s3URI    string
		region   string
		expected string
	}{
		{
			s3URI:    "s3://my-bucket/path/to/obj.txt",
			region:   "us-east-1",
			expected: "https://s3.us-east-1.amazonaws.com/my-bucket/path/to/obj.txt",
		},
		{
			s3URI:    "s3://simple-bucket/file.md",
			region:   "ap-northeast-1",
			expected: "https://s3.ap-northeast-1.amazonaws.com/simple-bucket/file.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.s3URI, func(t *testing.T) {
			got := convertS3URIToPublicURL(tt.s3URI, tt.region)
			if got != tt.expected {
				t.Errorf("convertS3URIToPublicURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}
