// git-gemini-cli は、Git差分ベースのAIコードレビューを行うCLIツールです。
package main

import (
	"github.com/shouni/git-gemini-cli/cmd" // CLIのエントリポイント
)

func main() {
	// cmd.Execute() を呼び出してアプリケーションを起動します。
	cmd.Execute()
}
