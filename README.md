# 🤖 Git Gemini CLI

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/git-gemini-cli)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/git-gemini-cli)](https://github.com/shouni/git-gemini-cli/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - 開発効率をブーストする、軽量AIレビューCLI

**Git Gemini CLI** は、AIコードレビューの**コアエンジン**を提供する **[Gemini Reviewer Core](https://github.com/shouni/gemini-reviewer-core)** を利用し、その機能をコマンドラインインターフェース（CLI）として実行可能にしたアプリケーションです。

---

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ | 役割 |
| :--- | :--- | :--- |
| **言語** | **Go (Golang)** | ツールの開発言語。クロスプラットフォームでの高速な実行を実現します。 |
| **CLI フレームワーク** | **Cobra** | コマンドライン引数（フラグ）の解析とサブコマンド構造 (`generic`, `publish`) の構築に使用します。 |
| **コアレビュー機能** | **[`gemini-reviewer-core`](https://github.com/shouni/gemini-reviewer-core)** | **Git操作、AI通信、HTML変換**といった中核のレビューロジックを担う外部ライブラリです。 |

---

## 🔄 実行ワークフロー (Execution Workflow)

* **分析フェーズ**: 指定されたブランチ間の差分（Diff）を取得し、`gemini-reviewer-core` を用いて AI によるコードレビューを実施します。
* **出力フェーズ**: レビュー結果（Markdown）を HTML に変換し、指定されたストレージ（ローカルまたは GCS）へ保存します。
* **通知フェーズ**: 公開準備が整い次第、Slack 等の外部サービスへレビュー完了レポートを即座に通知します。

---

## 🎨 概要イメージ

![Page 1](./docs/manga_page_1.png)
![Page 2](./docs/manga_page_2.png)

---

## 🛠️ 事前準備と環境設定

### 1\. プロジェクトのセットアップとビルド

```bash
# リポジトリをクローン
git clone git@github.com:shouni/git-gemini-cli.git

# 実行ファイルを bin/ ディレクトリに生成
go build -o bin/git_gemini_cli
```

実行ファイルは、プロジェクトルートの **`./bin/git_gemini_cli`** に生成されます。

---

### 2\. 環境変数の設定 (必須)

Gemini API を利用するために、API キーを環境変数に設定する必要があります。また、連携サービスを使用する場合は、対応する環境変数を設定します。

```bash
# GCP プロジェクトID
export GCP_PROJECT_ID="YOUR_GCP_PROJECT_ID"
# Gemini API キー
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
# Slack 連携 (publishモードで保存成功時に公開URLが通知されます)
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
```

---

## 🤖 プロンプト設定とAIコードレビューの種類 (`--mode` オプション)

本ツールは、レビューの目的に応じて AI に与える指示（**プロンプト**）を切り替えることができます。これは共通フラグの **`-m`, `--mode`** で指定します。

| モード (`-m`) | プロンプトファイル | 目的とレビュー観点 |
| :--- | :--- | :--- |
| **`detail`** | **[`assets/prompts/prompt_detail.md`](assets/prompts/prompt_detail.md)** | **コード品質と保守性の向上**を目的とした詳細なレビュー。可読性、重複、命名規則、一般的なベストプラクティスからの逸脱など、広範囲な技術的側面に焦点を当てます。 |
| **`release`** | **[`assets/prompts/prompt_release.md`](assets/prompts/prompt_release.md)** | **本番リリース可否の判定**を目的としたクリティカルなレビュー。致命的なバグ、セキュリティ脆弱性、サーバーダウンにつながる重大なパフォーマンス問題など、リリースをブロックする問題に限定して指摘します。 |

---

## 🚀 使い方 (Usage) と実行例

このツールは、**リモートリポジトリのブランチ間比較**に特化しており、**サブコマンド**を使用します。

### 🛠 共通フラグ (Persistent Flags)

すべてのサブコマンド (`generic`, `publish`) で使用可能なフラグです。

| フラグ | ショートカット | 説明 | デフォルト値 | 必須 |
| :--- | :--- | :--- | :--- | :--- |
| `--mode` | **`-m`** | レビューモードを指定: `'release'` (リリース判定) または `'detail'` (詳細レビュー) | `detail` | ❌ |
| `--repo-url` | **`-u`** | レビュー対象の Git リポジトリの **SSH URL** | **なし** | ✅ |
| `--base-branch` | **`-b`** | 基準となる**ブランチ名またはハッシュ** | `main` | ❌ |
| `--feature-branch` | **`-f`** | レビュー対象の**ブランチ名またはハッシュ** | **なし** | ✅ |
| `--gemini` | **`-g`** | 使用する Gemini モデル名 (例: `gemini-2.5-flash`) | `gemini-2.5-flash` | ❌ |
| `--ssh-key-path` | **`-k`** | Git 認証用の SSH 秘密鍵のパス。**チルダ (`~`) 展開をサポート**しています。**CI/CD環境ではシークレットマウント先の絶対パス**を指定してください。 | `~/.ssh/id_rsa` | ❌ |
| `--skip-host-key-check` | なし | SSHホストキーチェックをスキップする（**🚨非推奨/危険な設定**）。**`known_hosts`を使用しない**場合に設定します。 | `false` | ❌ |
| `--use-external-git` | なし | ローカルのGitコマンド使用する。 | **`true`** | ❌ |

---

### 1\. 標準出力モード (`generic`)

リモートリポジトリのブランチ差分を取得し、レビュー結果を**標準出力**に出力します。

#### 実行コマンド例

```bash
# main と develop の差分をリリース判定モードで実行
./bin/git_gemini_cli generic \
  -m "release" \
  --repo-url "git@github.com:user/my-awesome-project.git" \
  --base-branch "main" \
  --feature-branch "develop"
```

---

### 2. クラウド保存モード (`publish`) 

リモートリポジトリのブランチ比較を行い、その結果を **GCS（Google Cloud Storage）** などのクラウドストレージに、**スタイル付き HTML** として保存します。保存完了後、チームへの共有を自動化するための通知機能も備えています。

**💡 Slack通知について:**
`SLACK_WEBHOOK_URL` が設定されている場合、保存成功後に **GCS上の保存先URI（または署名付きURL）** を含めたレビュー完了レポートをSlackへ自動投稿します。

#### 実行コマンド例 (GCSへの保存)

```bash
# feature/publish の差分をレビューし、GCSにHTML結果を保存
./bin/git_gemini_cli publish \
  -m "detail" \
  --repo-url "git@github.com:user/my-awesome-project.git" \
  --base-branch "main" \
  --feature-branch "feature/publish" \
  --bucket "my-awesome-bucket"
```

#### 固有フラグ (クラウド連携)

| フラグ | ショートカット | 説明 | 必須 | デフォルト値 |
| :--- | :--- | :--- | :--- | :--- |
| `--bucket` | なし | 保存先の GCS バケット名 | ✅ | **なし** |

---

### 📜 ライセンス (License)

* デフォルトキャラクター: VOICEVOX:ずんだもん、VOICEVOX:四国めたん
* このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。

---