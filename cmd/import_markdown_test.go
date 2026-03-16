package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/riba2534/feishu-cli/internal/config"
	"github.com/spf13/viper"
)

func initImportCommandTestConfig(t *testing.T) {
	t.Helper()

	viper.Reset()
	t.Setenv("FEISHU_APP_ID", "cli_xxx")
	t.Setenv("FEISHU_APP_SECRET", "xxx")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("写入测试配置文件失败: %v", err)
	}
	if err := config.Init(configPath); err != nil {
		t.Fatalf("初始化测试配置失败: %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道失败: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("读取标准输出失败: %v", err)
	}
	return buf.String()
}

func TestNormalizeImportMode(t *testing.T) {
	tests := []struct {
		name            string
		documentID      string
		mode            string
		replaceExisting bool
		appendExisting  bool
		want            string
		wantErr         bool
	}{
		{
			name: "新建文档默认模式为空",
			want: "",
		},
		{
			name:       "已有文档默认覆盖",
			documentID: "doc123",
			want:       importModeReplace,
		},
		{
			name:       "已有文档显式覆盖",
			documentID: "doc123",
			mode:       importModeReplace,
			want:       importModeReplace,
		},
		{
			name:       "已有文档显式追加",
			documentID: "doc123",
			mode:       importModeAppend,
			want:       importModeAppend,
		},
		{
			name:            "replace 别名切到覆盖",
			documentID:      "doc123",
			replaceExisting: true,
			want:            importModeReplace,
		},
		{
			name:           "append 别名切到追加",
			documentID:     "doc123",
			appendExisting: true,
			want:           importModeAppend,
		},
		{
			name:    "新建文档不能指定模式",
			mode:    importModeReplace,
			wantErr: true,
		},
		{
			name:            "新建文档不能指定 replace",
			replaceExisting: true,
			wantErr:         true,
		},
		{
			name:           "新建文档不能指定 append",
			appendExisting: true,
			wantErr:        true,
		},
		{
			name:            "append 模式与 replace 别名冲突",
			documentID:      "doc123",
			mode:            importModeAppend,
			replaceExisting: true,
			wantErr:         true,
		},
		{
			name:           "replace 模式与 append 别名冲突",
			documentID:     "doc123",
			mode:           importModeReplace,
			appendExisting: true,
			wantErr:        true,
		},
		{
			name:            "append 与 replace 别名冲突",
			documentID:      "doc123",
			replaceExisting: true,
			appendExisting:  true,
			wantErr:         true,
		},
		{
			name:       "非法模式报错",
			documentID: "doc123",
			mode:       "rewrite",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportMode(tt.documentID, tt.mode, tt.replaceExisting, tt.appendExisting)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望返回错误")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeImportMode() 返回错误: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeImportMode() = %q, 期望 %q", got, tt.want)
			}
		})
	}
}

func TestRunImportMarkdown_DefaultDocumentIDModeDispatchesToReplace(t *testing.T) {
	initImportCommandTestConfig(t)

	filePath := filepath.Join(t.TempDir(), "replace.md")
	if err := os.WriteFile(filePath, []byte("# 替换内容\n"), 0600); err != nil {
		t.Fatalf("写入测试 Markdown 失败: %v", err)
	}

	oldReplaceFn := runReplaceContentFn
	oldImportFn := importMarkdownIntoParentFn
	defer func() {
		runReplaceContentFn = oldReplaceFn
		importMarkdownIntoParentFn = oldImportFn
	}()

	var called bool
	var got replaceContentRequest
	runReplaceContentFn = func(req replaceContentRequest) error {
		called = true
		got = req
		return nil
	}
	importMarkdownIntoParentFn = func(documentID string, parentBlockID string, markdownText string, basePath string, uploadImages bool, verbose bool, diagramWorkers int, tableWorkers int, diagramRetries int) (*importStats, error) {
		t.Fatalf("覆盖模式不应直接调用 importMarkdownIntoParent")
		return nil, nil
	}

	err := runImportMarkdown(importMarkdownRequest{
		filePath:       filePath,
		documentID:     "doc_replace_123",
		uploadImages:   true,
		verbose:        true,
		output:         "json",
		diagramWorkers: 7,
		tableWorkers:   4,
		diagramRetries: 11,
	})
	if err != nil {
		t.Fatalf("runImportMarkdown() 返回错误: %v", err)
	}
	if !called {
		t.Fatalf("期望调用 runReplaceContentFn")
	}
	if got.documentID != "doc_replace_123" {
		t.Fatalf("documentID = %q, 期望 %q", got.documentID, "doc_replace_123")
	}
	if got.source != filePath {
		t.Fatalf("source = %q, 期望 %q", got.source, filePath)
	}
	if got.sourceType != "file" {
		t.Fatalf("sourceType = %q, 期望 file", got.sourceType)
	}
	if got.parentBlockID != "doc_replace_123" {
		t.Fatalf("parentBlockID = %q, 期望 %q", got.parentBlockID, "doc_replace_123")
	}
	if !got.force {
		t.Fatalf("覆盖模式应强制走无交互替换")
	}
	if !got.skipValidation {
		t.Fatalf("覆盖模式复用时应跳过重复校验")
	}
	if !got.verbose {
		t.Fatalf("verbose 应传递给替换逻辑")
	}
	if got.diagramWorkers != 7 || got.tableWorkers != 4 || got.diagramRetries != 11 {
		t.Fatalf("并发参数未正确透传: %+v", got)
	}
}

func TestRunImportMarkdown_AppendModeAppends(t *testing.T) {
	initImportCommandTestConfig(t)

	filePath := filepath.Join(t.TempDir(), "append.md")
	content := "# 追加内容\n\n正文"
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("写入测试 Markdown 失败: %v", err)
	}

	oldReplaceFn := runReplaceContentFn
	oldImportFn := importMarkdownIntoParentFn
	defer func() {
		runReplaceContentFn = oldReplaceFn
		importMarkdownIntoParentFn = oldImportFn
	}()

	runReplaceContentFn = func(req replaceContentRequest) error {
		t.Fatalf("追加模式不应走替换逻辑")
		return nil
	}

	var called bool
	var gotDocumentID string
	var gotParentBlockID string
	var gotContent string
	var gotBasePath string
	var gotUploadImages bool
	var gotVerbose bool
	var gotDiagramWorkers int
	var gotTableWorkers int
	var gotDiagramRetries int

	importMarkdownIntoParentFn = func(documentID string, parentBlockID string, markdownText string, basePath string, uploadImages bool, verbose bool, diagramWorkers int, tableWorkers int, diagramRetries int) (*importStats, error) {
		called = true
		gotDocumentID = documentID
		gotParentBlockID = parentBlockID
		gotContent = markdownText
		gotBasePath = basePath
		gotUploadImages = uploadImages
		gotVerbose = verbose
		gotDiagramWorkers = diagramWorkers
		gotTableWorkers = tableWorkers
		gotDiagramRetries = diagramRetries
		return &importStats{}, nil
	}

	_ = captureStdout(t, func() {
		err := runImportMarkdown(importMarkdownRequest{
			filePath:       filePath,
			documentID:     "doc_append_123",
			uploadImages:   true,
			verbose:        true,
			output:         "json",
			diagramWorkers: 5,
			tableWorkers:   3,
			diagramRetries: 9,
			appendExisting: true,
		})
		if err != nil {
			t.Fatalf("runImportMarkdown() 返回错误: %v", err)
		}
	})

	if !called {
		t.Fatalf("期望调用 importMarkdownIntoParentFn")
	}
	if gotDocumentID != "doc_append_123" || gotParentBlockID != "doc_append_123" {
		t.Fatalf("文档 ID 传递错误: documentID=%q parentBlockID=%q", gotDocumentID, gotParentBlockID)
	}
	if gotContent != content {
		t.Fatalf("markdown 内容不匹配: got=%q want=%q", gotContent, content)
	}
	if gotBasePath != filepath.Dir(filePath) {
		t.Fatalf("basePath = %q, 期望 %q", gotBasePath, filepath.Dir(filePath))
	}
	if !gotUploadImages || !gotVerbose {
		t.Fatalf("布尔参数未正确透传")
	}
	if gotDiagramWorkers != 5 || gotTableWorkers != 3 || gotDiagramRetries != 9 {
		t.Fatalf("并发参数未正确透传")
	}
}
