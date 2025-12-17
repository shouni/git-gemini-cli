package config

import (
	"reflect"
	"strings"

	"github.com/shouni/go-http-kit/pkg/httpkit"
)

// ReviewConfig はAIコードレビューに必要なすべての設定を含みます。
// この構造体は、コマンドライン引数からサービスロジックへ設定を渡すための共通のデータモデルです。
type ReviewConfig struct {
	ReviewMode            string
	GeminiModel           string
	RepoURL               string
	BaseBranch            string
	FeatureBranch         string
	SSHKeyPath            string
	LocalPath             string
	SkipHostKeyCheck      bool
	UseExternalGitCommand bool
}

type PublishConfig struct {
	HttpClient      httpkit.ClientInterface
	ReviewConfig    ReviewConfig
	StorageURI      string
	SlackWebhookURL string
}

// Normalize はリフレクションを用いて設定値の文字列フィールドから前後の空白を一括で削除します。
func (c *ReviewConfig) Normalize() {
	v := reflect.ValueOf(c).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		// フィールドが文字列型で、かつ設定可能な場合のみ処理
		if field.Type().Kind() == reflect.String && field.CanSet() {
			trimmedValue := strings.TrimSpace(field.String())
			field.SetString(trimmedValue)
		}
	}
}
