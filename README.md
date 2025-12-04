# 🤖 Git Gemini Cli

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/git-gemini-cli)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/git-gemini-cli)](https://github.com/shouni/git-gemini-cli/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - 開発チームの生産性を高めるAIパートナー

**Git Gemini Cli** は、AIコードレビューの**コア機能**を **[Gemini Reviewer Core](https://github.com/shouni/gemini-reviewer-core)** モジュールに依存し、それをCLIとして公開するための**ラッパーアプリケーション**です。

本ツールは、ユーザーの入力（CLIフラグ）を解析し、コアライブラリを組み合わせて**レビューパイプライン全体を実行**し、結果を様々なサービス（クラウドストレージ, 標準出力）に投稿する**インターフェースの責務**を担います。AIは煩雑な初期チェックを担う、**チームの優秀な新しいパートナー**のような存在です。

-----

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ | 役割 |
| :--- | :--- | :--- |
| **言語** | **Go (Golang)** | ツールの開発言語。クロスプラットフォームでの高速な実行を実現します。 |
| **CLI フレームワーク** | **Cobra** | コマンドライン引数（フラグ）の解析とサブコマンド構造 (`generic`, `backlog`, `slack`, `publish`) の構築に使用します。 |
| **コアロジック** | **`github.com/shouni/gemini-reviewer-core`** | **Git操作**、**AI通信**、**HTML変換**といった中核のレビュー機能を担う外部ライブラリです。 |
| **AI通信** | **`google.golang.org/genai` (Go SDK)** | Gemini APIへのアクセス。リトライ機構付きでSDKをラッピングし、堅牢な通信を実現します。 |
| **ロギング** | **log/slog** | 構造化されたログ (`key=value`) に完全移行。詳細なデバッグ情報が必要な際に、ログレベルを上げて柔軟に対応できます。 |

-----

## 🧩 アーキテクチャ設計と採用理由 (Local Optimized)

本ツールは、**ローカル環境での高速実行**と**既存のGit設定とのシームレスな統合**を目的として、**`os/exec`** を使用したローカルGitコマンド実行です。

| 特徴 | **本ツール (CLI Tool)** (現行設計) | **Web Runner** (設計) |
| :--- | :--- | :--- |
| **Git操作** | **外部コマンド実行 (`os/exec`)**<br>OSの `git diff` を直接使用。**高速**で `.gitattributes` 等も考慮される。 | **純粋な Go 実装 (`go-git`)**<br>OS非依存だが、大規模リポジトリでは遅延やメモリ消費の可能性。 |
| **更新戦略** | **Pull 主体 (永続化)**<br>ローカルリポジトリを維持し、`git pull` でワーキングツリーを更新。 | **Fetch 主体 (使い捨て)**<br>`Pull` せず `Fetch` でDBのみ更新し、毎回クリーンアップする。 |
| **SSH認証** | **OS設定を利用**<br>`~/.ssh/config` や `ssh-agent`、**`GIT_SSH_COMMAND`** を利用し、ユーザーの既存設定で認証する。 | **Go内で完結**<br>秘密鍵を読み込み、プログラム内で署名を行う。 |
| **Context** | **あり** (`exec.CommandContext`を使用)<br>実行フローは同期的ながら、**タイムアウト制御**と**中断処理**をサポート。 | **あり (必須)** |

-----

## 🛠️ 事前準備と環境設定

### 1\. プロジェクトのセットアップとビルド

```bash
# リポジトリをクローン
git clone git@github.com:shouni/git-gemini-cli.git

# 実行ファイルを bin/ ディレクトリに生成
go build -o bin/git_gemini_cli
```

実行ファイルは、プロジェクトルートの **`./bin/git_gemini_cli`** に生成されます。

-----

### 3\. 環境変数の設定 (必須)

Gemini API を利用するために、API キーを環境変数に設定する必要があります。また、連携サービスを使用する場合は、対応する環境変数を設定します。

```bash
# Gemini API キー (必須)
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
# Slack 連携
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
```

-----

### 4\. モデルパラメータとプロンプト設定について (重要) 🆕

本ツールが使用する以下のコア設定は、依存先の **[`gemini-reviewer-core`](https://github.com/shouni/gemini-reviewer-core)** モジュール内で定義・管理されています。

* **温度 (Temperature):** `0.1` に設定されています。
    * この低い温度設定は、応答の安定性を優先し、一貫性のあるコードレビュー結果を生成するために、コアライブラリ側で適用されています。
* **プロンプト設定:** プロンプトテンプレートファイル (`.md`) は、**コアライブラリのリポジトリ**に配置されており、本ツールでは**変更できません**。内容を確認・変更したい場合は、[`gemini-reviewer-core` ](https://github.com/shouni/gemini-reviewer-core) のリポジトリを参照してください。

-----

## 🤖 AIコードレビューの種類 (`--mode` オプション)

本ツールは、レビューの目的に応じて AI に与える指示（**プロンプト**）を切り替えることができます。これは共通フラグの **`-m`, `--mode`** で指定します。

| モード (`-m`) | プロンプトファイル (参照先) | 目的とレビュー観点 |
| :--- | :--- | :--- |
| **`detail`** | **gemini-reviewer-core/prompts/prompt\_detail.md** | **コード品質と保守性の向上**を目的とした詳細なレビュー。可読性、重複、命名規則、一般的なベストプラクティスからの逸脱など、広範囲な技術的側面に焦点を当てます。 |
| **`release`** | **gemini-reviewer-core/prompts/prompt\_release.md** | **本番リリース可否の判定**を目的としたクリティカルなレビュー。致命的なバグ、セキュリティ脆弱性、サーバーダウンにつながる重大なパフォーマンス問題など、リリースをブロックする問題に限定して指摘します。 |

-----

## 🚀 使い方 (Usage) と実行例

このツールは、**リモートリポジトリのブランチ間比較**に特化しており、**サブコマンド**を使用します。

### 🛠 共通フラグ (Persistent Flags)

すべてのサブコマンド (`generic`, `backlog`, `slack`, `publish`) で使用可能なフラグです。

| フラグ | ショートカット | 説明 | デフォルト値 | 必須 |
| :--- | :--- | :--- | :--- | :--- |
| `--mode` | **`-m`** | レビューモードを指定: `'release'` (リリース判定) または `'detail'` (詳細レビュー) | `detail` | ❌ |
| `--repo-url` | **`-u`** | レビュー対象の Git リポジトリの **SSH URL** | **なし** | ✅ |
| `--base-branch` | **`-b`** | 差分比較の基準ブランチ | `main` | ❌ |
| `--feature-branch` | **`-f`** | レビュー対象のフィーチャーブランチ | **なし** | ✅ |
| `--local-path` | **`-l`** | リポジトリをクローンするローカルパス | 一時ディレクトリ | ❌ |
| `--gemini` | **`-g`** | 使用する Gemini モデル名 (例: `gemini-2.5-flash`) | `gemini-2.5-flash` | ❌ |
| `--ssh-key-path` | **`-k`** | Git 認証用の SSH 秘密鍵のパス。**チルダ (`~`) 展開をサポート**しています。**CI/CD環境ではシークレットマウント先の絶対パス**を指定してください。 | `~/.ssh/id_rsa` | ❌ |
| `--skip-host-key-check` | なし | SSHホストキーチェックをスキップする（**🚨非推奨/危険な設定**）。**`known_hosts`を使用しない**場合に設定します。 | `false` | ❌ |
| **`--use-internal-git-adapter`** | なし | ローカルのGitコマンド使用する。 | **`true`** | ❌ |

-----

### 1\. 標準出力モード (`generic`)

リモートリポジトリのブランチ差分を取得し、レビュー結果を**標準出力**に出力します。

#### 実行コマンド例

```bash
# main と develop の差分をリリース判定モードで実行
./bin/git_gemini_cli generic \
  -m "release" \
  --repo-url "git@example.backlog.jp:PROJECT/repo-name.git" \
  --base-branch "main" \
  --feature-branch "develop"
```

-----

### 2\. クラウド保存モード (`publish`) 🌟 (マルチクラウド対応)

リモートリポジトリのブランチ比較を行い、その結果を **URI で指定されたクラウドストレージ（GCSまたはS3）** に、**AIが出力したMarkdownを専用ライブラリ（go-text-format）で変換したスタイル付き HTML** として保存します。このモードは、レビュー結果のアーカイブや、CI/CDパイプラインでのレポート生成を目的としています。

#### 実行コマンド例 (GCSへの保存)

```bash
# feature/publish の差分をレビューし、GCSにHTML結果を保存
./bin/git_gemini_cli publish \
  -m "detail" \
  --repo-url "git@example.backlog.jp:PROJECT/repo-name.git" \
  --base-branch "main" \
  --feature-branch "feature/publish" \
  --uri "gs://review-archive-bucket/reviews/2025/latest_review.html" 
```

#### 実行コマンド例 (S3への保存)

```bash
# feature/s3-save の差分をレビューし、S3にHTML結果を保存
./bin/git_gemini_cli publish \
  -m "release" \
  --repo-url "git@example.backlog.jp:PROJECT/repo-name.git" \
  --base-branch "main" \
  --feature-branch "feature/s3-save" \
  --uri "s3://review-report-bucket/reports/2025/latest_release.html" 
```

#### 固有フラグ (クラウド連携)

| フラグ | ショートカット | 説明 | 必須 | デフォルト値 |
| :--- | :--- | :--- | :--- | :--- |
| `--uri` | **`-s`** | 書き込み先 URI (**`gs://...`** または **`s3://...`** をサポート) | ✅ | **なし** |
| `--content-type` | **`-t`** | クラウドストレージに保存するファイルのMIMEタイプ | ❌ | **`text/html; charset=utf-8`** |

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
