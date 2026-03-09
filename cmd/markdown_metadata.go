package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var markdownFrontMatterPattern = regexp.MustCompile(`(?s)^---[ \t]*\r?\n(.*?)\r?\n(?:---|\.\.\.)[ \t]*(?:\r?\n|$)`)

// markdownMetadata 表示 Markdown 顶部的 YAML 元数据。
type markdownMetadata struct {
	Title      string `yaml:"title,omitempty"`
	DocumentID string `yaml:"document_id,omitempty"`
	NodeToken  string `yaml:"node_token,omitempty"`
}

// parsedMarkdownDocument 表示解析后的 Markdown 文档。
type parsedMarkdownDocument struct {
	Metadata       markdownMetadata
	Body           string
	HasFrontMatter bool
}

// parseMarkdownDocument 解析 YAML front matter；无法识别时按普通 Markdown 处理。
func parseMarkdownDocument(content string) parsedMarkdownDocument {
	normalized := strings.TrimPrefix(content, "\uFEFF")
	matches := markdownFrontMatterPattern.FindStringSubmatchIndex(normalized)
	if matches == nil {
		return parsedMarkdownDocument{Body: normalized}
	}

	rawFrontMatter := normalized[matches[2]:matches[3]]
	var fields map[string]any
	if err := yaml.Unmarshal([]byte(rawFrontMatter), &fields); err != nil || len(fields) == 0 {
		return parsedMarkdownDocument{Body: normalized}
	}

	return parsedMarkdownDocument{
		Metadata: markdownMetadata{
			Title:      frontMatterScalar(fields["title"]),
			DocumentID: frontMatterScalar(fields["document_id"]),
			NodeToken:  frontMatterScalar(fields["node_token"]),
		},
		Body:           normalized[matches[1]:],
		HasFrontMatter: true,
	}
}

func frontMatterScalar(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bool:
		return fmt.Sprint(v)
	default:
		return ""
	}
}

func defaultDocumentTitleFromPath(filePath string) string {
	if strings.TrimSpace(filePath) == "" {
		return "无标题文档"
	}

	title := filepath.Base(filePath)
	if title == "." {
		return "无标题文档"
	}
	ext := filepath.Ext(title)
	if len(ext) < len(title) {
		title = title[:len(title)-len(ext)]
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return "无标题文档"
	}
	return title
}

func resolveImportDocumentTitle(cliTitle string, metadata markdownMetadata, filePath string) string {
	if title := strings.TrimSpace(cliTitle); title != "" {
		return title
	}
	if title := strings.TrimSpace(metadata.Title); title != "" {
		return title
	}
	return defaultDocumentTitleFromPath(filePath)
}

func prependMarkdownFrontMatter(markdown string, metadata markdownMetadata) (string, error) {
	if metadata == (markdownMetadata{}) {
		return markdown, nil
	}

	data, err := yaml.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("序列化 YAML front matter 失败: %w", err)
	}
	return fmt.Sprintf("---\n%s---\n\n%s", string(data), markdown), nil
}
