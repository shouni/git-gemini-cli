package adapters

import (
	"context"
	"fmt"

	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-remote-io/pkg/s3factory"
)

// InitPublisherAndSigner は、URIに基づいてPublisherとURLSignerを初期化します。
func InitPublisherAndSigner(ctx context.Context, targetURI string) (publisher.Publisher, remoteio.URLSigner, error) {
	registry := publisher.FactoryRegistry{}
	var urlSigner remoteio.URLSigner
	var err error

	// GCSまたはS3のどちらか必要なファクトリのみを初期化し、RegistryとSignerを設定
	if remoteio.IsGCSURI(targetURI) {
		gcsFactory, err := gcsfactory.NewGCSClientFactory(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("GCSクライアントファクトリの初期化に失敗しました: %w", err)
		}
		registry.GCSFactory = gcsFactory

		signer, err := gcsFactory.NewGCSURLSigner()
		if err != nil {
			return nil, nil, fmt.Errorf("GCS URL Signerの取得に失敗しました: %w", err)
		}
		urlSigner = signer

	} else if remoteio.IsS3URI(targetURI) {
		s3Factory, err := s3factory.NewS3ClientFactory(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("S3クライアントファクトリの初期化に失敗しました (URI: %s): %w", targetURI, err)
		}
		registry.S3Factory = s3Factory

		signer, err := s3Factory.NewS3URLSigner()
		if err != nil {
			return nil, nil, fmt.Errorf("S3 URL Signerの取得に失敗しました: %w", err)
		}
		urlSigner = signer

	} else {
		return nil, nil, fmt.Errorf("未対応のストレージURIです: %s", targetURI)
	}

	// Publisherの動的生成
	writer, err := publisher.NewPublisher(targetURI, registry)
	if err != nil {
		// Publisher.NewPublisherでURIスキームがサポート外の場合もここでエラーになる
		return nil, nil, fmt.Errorf("パブリッシャーの初期化に失敗しました: %w", err)
	}

	return writer, urlSigner, nil
}
