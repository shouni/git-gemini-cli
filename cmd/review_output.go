package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
)

// reviewOutput は、gemini-reviewer-core の ai.reviewOutputSchema に対応する構造です。
// AIレビュー結果JSONを標準出力向けに整形するためだけに使う内部実装の詳細です。
type reviewOutput struct {
	Title    string    `json:"title"`
	Summary  string    `json:"summary"`
	Verdict  verdict   `json:"verdict"`
	Findings []finding `json:"findings"`
}

type verdict struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type finding struct {
	Severity   string `json:"severity"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Excerpt    string `json:"excerpt"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// formatReviewOutput は、AIレビュー結果JSONを標準出力向けの読みやすいテキストに整形します。
func formatReviewOutput(content string) (string, error) {
	var out reviewOutput
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return "", fmt.Errorf("レビュー結果JSONのパースに失敗: %w", err)
	}

	var b strings.Builder

	if out.Title != "" {
		fmt.Fprintf(&b, "■ %s\n\n", out.Title)
	}

	if out.Verdict.Decision != "" {
		fmt.Fprintf(&b, "判定: %s", out.Verdict.Decision)
		if out.Verdict.Reason != "" {
			fmt.Fprintf(&b, " — %s", out.Verdict.Reason)
		}
		b.WriteString("\n\n")
	}

	if out.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", out.Summary)
	}

	if len(out.Findings) > 0 {
		b.WriteString("--- 指摘事項 ---\n")
		for i, f := range out.Findings {
			location := f.File
			if f.Line > 0 {
				location = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			fmt.Fprintf(&b, "\n[%d] [%s] %s\n", i+1, f.Severity, location)
			if f.Excerpt != "" {
				fmt.Fprintf(&b, "  引用: %s\n", f.Excerpt)
			}
			fmt.Fprintf(&b, "  %s\n", f.Message)
			if f.Suggestion != "" {
				fmt.Fprintf(&b, "  修正案: %s\n", f.Suggestion)
			}
		}
	}

	return strings.TrimRight(b.String(), "\n"), nil
}
