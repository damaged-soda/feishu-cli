package cmd

import "testing"

func TestParseMarkdownDocument_WithFrontMatter(t *testing.T) {
	input := "---\n" +
		"title: 元数据标题\n" +
		"document_id: doc_123\n" +
		"node_token: wiki_456\n" +
		"---\n\n" +
		"# 正文标题\n正文内容\n"

	got := parseMarkdownDocument(input)
	if !got.HasFrontMatter {
		t.Fatalf("期望识别 front matter")
	}
	if got.Metadata.Title != "元数据标题" {
		t.Fatalf("title = %q, 期望 %q", got.Metadata.Title, "元数据标题")
	}
	if got.Metadata.DocumentID != "doc_123" {
		t.Fatalf("document_id = %q, 期望 %q", got.Metadata.DocumentID, "doc_123")
	}
	if got.Metadata.NodeToken != "wiki_456" {
		t.Fatalf("node_token = %q, 期望 %q", got.Metadata.NodeToken, "wiki_456")
	}
	wantBody := "\n# 正文标题\n正文内容\n"
	if got.Body != wantBody {
		t.Fatalf("body = %q, 期望 %q", got.Body, wantBody)
	}
}

func TestParseMarkdownDocument_FallbackToPlainMarkdown(t *testing.T) {
	input := "---\n这不是 front matter\n---\n\n# 正文标题\n"

	got := parseMarkdownDocument(input)
	if got.HasFrontMatter {
		t.Fatalf("不应将普通 Markdown 误识别为 front matter")
	}
	if got.Body != input {
		t.Fatalf("body = %q, 期望保持原文 %q", got.Body, input)
	}
}

func TestResolveImportDocumentTitle(t *testing.T) {
	tests := []struct {
		name     string
		cliTitle string
		metadata markdownMetadata
		filePath string
		want     string
	}{
		{
			name:     "命令行标题优先级最高",
			cliTitle: "命令行标题",
			metadata: markdownMetadata{Title: "front matter 标题"},
			filePath: "/tmp/demo.md",
			want:     "命令行标题",
		},
		{
			name:     "front matter 标题次高",
			metadata: markdownMetadata{Title: "front matter 标题"},
			filePath: "/tmp/demo.md",
			want:     "front matter 标题",
		},
		{
			name:     "回落到文件名",
			filePath: "/tmp/demo.md",
			want:     "demo",
		},
		{
			name:     "空文件名回落到默认标题",
			filePath: "",
			want:     "无标题文档",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveImportDocumentTitle(tt.cliTitle, tt.metadata, tt.filePath)
			if got != tt.want {
				t.Fatalf("resolveImportDocumentTitle() = %q, 期望 %q", got, tt.want)
			}
		})
	}
}

func TestPrependMarkdownFrontMatter_PreservesBodyHeading(t *testing.T) {
	markdown := "# 正文标题\n正文内容\n"
	got, err := prependMarkdownFrontMatter(markdown, markdownMetadata{
		Title:      "文档标题",
		DocumentID: "doc_123",
	})
	if err != nil {
		t.Fatalf("prependMarkdownFrontMatter() 返回错误: %v", err)
	}

	want := "---\n" +
		"title: 文档标题\n" +
		"document_id: doc_123\n" +
		"---\n\n" +
		"# 正文标题\n正文内容\n"
	if got != want {
		t.Fatalf("prependMarkdownFrontMatter() = %q, 期望 %q", got, want)
	}
}
