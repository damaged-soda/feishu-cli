package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/riba2534/feishu-cli/internal/client"
	"github.com/riba2534/feishu-cli/internal/config"
	"github.com/spf13/cobra"
)

type replaceContentRequest struct {
	documentID     string
	source         string
	sourceType     string
	parentBlockID  string
	allowEmpty     bool
	force          bool
	uploadImages   bool
	diagramWorkers int
	tableWorkers   int
	diagramRetries int
	output         string
	verbose        bool
	skipValidation bool
}

var runReplaceContentFn = runReplaceContent

var replaceContentCmd = &cobra.Command{
	Use:   "replace <document_id> [source]",
	Short: "用 Markdown 替换父块下的全部子块",
	Long: `用 Markdown 内容替换飞书文档中某个父块下的全部直接子块。

默认替换文档根节点下的全部内容，也可以通过 --block-id 指定父块。

为降低误清空风险，命令默认采用“先创建新内容，再删除旧内容”的策略：
1. 先将新 Markdown 追加到目标父块末尾
2. 新内容全部创建成功后，再删除旧内容
3. 如果中途失败，命令会尽力清理刚追加的新内容，并保留旧内容

参数:
  <document_id>        文档 ID（必填）
  [source]             Markdown 文件路径（与 --content / --content-file 三选一）
  --content, -c        Markdown 内容字符串
  --content-file       Markdown 文件路径
  --source-type        源类型：file/content，默认 file
  --block-id, -b       目标父块 ID（默认: 文档根节点）
  --allow-empty        允许空内容，此时会清空目标父块下的全部子块
  --force, -f          跳过确认直接执行
  --upload-images      上传 Markdown 中的本地图片
  --diagram-workers    图表并发数
  --table-workers      表格并发数
  --diagram-retries    图表重试次数
  --output, -o         输出格式（json）

示例:
  feishu-cli doc replace DOC_ID README.md
  feishu-cli doc replace DOC_ID --content "# 新文档" --source-type content
  feishu-cli doc replace DOC_ID section.md --block-id BLOCK_ID
  feishu-cli doc replace DOC_ID --content "" --source-type content --allow-empty --force`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := replaceContentRequest{documentID: args[0]}
		contentStr, _ := cmd.Flags().GetString("content")
		contentFile, _ := cmd.Flags().GetString("content-file")
		req.sourceType, _ = cmd.Flags().GetString("source-type")
		req.parentBlockID, _ = cmd.Flags().GetString("block-id")
		req.allowEmpty, _ = cmd.Flags().GetBool("allow-empty")
		req.force, _ = cmd.Flags().GetBool("force")
		req.uploadImages, _ = cmd.Flags().GetBool("upload-images")
		req.diagramWorkers, _ = cmd.Flags().GetInt("diagram-workers")
		req.tableWorkers, _ = cmd.Flags().GetInt("table-workers")
		req.diagramRetries, _ = cmd.Flags().GetInt("diagram-retries")
		req.output, _ = cmd.Flags().GetString("output")

		if len(args) > 1 {
			req.source = args[1]
			if req.sourceType == "" {
				req.sourceType = "file"
			}
		} else if contentFile != "" {
			req.source = contentFile
			req.sourceType = "file"
		} else if contentStr != "" || cmd.Flags().Changed("content") {
			req.source = contentStr
			req.sourceType = "content"
		} else {
			return fmt.Errorf("必须指定源文件（第二个参数）、--content 或 --content-file")
		}

		return runReplaceContent(req)
	},
}

func runReplaceContent(req replaceContentRequest) error {
	if !req.skipValidation {
		if err := config.Validate(); err != nil {
			return err
		}
	}

	var contentData string
	basePath := ""
	if req.sourceType == "file" {
		basePath = filepath.Dir(req.source)
		data, err := os.ReadFile(req.source)
		if err != nil {
			return fmt.Errorf("读取 Markdown 文件失败: %w", err)
		}
		contentData = string(data)
	} else {
		contentData = req.source
	}

	if req.parentBlockID == "" {
		req.parentBlockID = req.documentID
	}

	oldChildren, err := client.GetAllBlockChildren(req.documentID, req.parentBlockID)
	if err != nil {
		return fmt.Errorf("获取目标父块子块失败: %w", err)
	}
	oldCount := len(oldChildren)

	trimmedContent := strings.TrimSpace(contentData)
	if trimmedContent == "" && !req.allowEmpty {
		return fmt.Errorf("输入内容为空；如需清空目标父块，请显式指定 --allow-empty")
	}

	if !req.force {
		prompt := fmt.Sprintf("确定要替换父块 %s 下的全部 %d 个子块吗？此操作会先写入新内容，再删除旧内容，且不可恢复", req.parentBlockID, oldCount)
		if !confirmAction(prompt) {
			fmt.Println("操作已取消")
			return nil
		}
	}

	if trimmedContent == "" {
		if oldCount > 0 {
			if err := client.DeleteBlocks(req.documentID, req.parentBlockID, 0, oldCount); err != nil {
				return fmt.Errorf("清空目标父块失败: %w", err)
			}
		}
		if req.output == "json" {
			return printJSON(map[string]any{
				"document_id":       req.documentID,
				"parent_block_id":   req.parentBlockID,
				"replaced_children": oldCount,
				"top_level_created": 0,
				"blocks":            0,
			})
		}
		fmt.Printf("替换成功！\n")
		fmt.Printf("  文档 ID: %s\n", req.documentID)
		fmt.Printf("  父块 ID: %s\n", req.parentBlockID)
		fmt.Printf("  替换前子块数: %d\n", oldCount)
		fmt.Printf("  新增顶层块数: 0\n")
		fmt.Printf("  总块数: 0\n")
		fmt.Printf("  链接: https://feishu.cn/docx/%s\n", req.documentID)
		return nil
	}

	stats, err := importMarkdownIntoParentFn(req.documentID, req.parentBlockID, contentData, basePath, req.uploadImages, req.verbose, req.diagramWorkers, req.tableWorkers, req.diagramRetries)
	if err != nil {
		cleanupErr := cleanupAppendedChildren(req.documentID, req.parentBlockID, oldCount)
		if cleanupErr != nil {
			return fmt.Errorf("替换失败: %w；同时清理新增内容失败: %v", err, cleanupErr)
		}
		return fmt.Errorf("替换失败，已尝试保留旧内容: %w", err)
	}

	currentChildren, err := client.GetAllBlockChildren(req.documentID, req.parentBlockID)
	if err != nil {
		return fmt.Errorf("获取替换后的子块失败: %w", err)
	}
	topLevelCreated := len(currentChildren) - oldCount
	if topLevelCreated < 0 {
		topLevelCreated = 0
	}

	if oldCount > 0 {
		if err := client.DeleteBlocks(req.documentID, req.parentBlockID, 0, oldCount); err != nil {
			cleanupErr := cleanupAppendedChildren(req.documentID, req.parentBlockID, oldCount)
			if cleanupErr != nil {
				return fmt.Errorf("删除旧内容失败: %w；同时清理新增内容失败: %v", err, cleanupErr)
			}
			return fmt.Errorf("删除旧内容失败，已尝试清理新增内容以保留旧内容: %w", err)
		}
	}

	if req.output == "json" {
		return printJSON(map[string]any{
			"document_id":       req.documentID,
			"parent_block_id":   req.parentBlockID,
			"replaced_children": oldCount,
			"top_level_created": topLevelCreated,
			"blocks":            stats.totalBlocks,
			"diagram_total":     stats.diagramTotal,
			"diagram_success":   stats.diagramSuccess,
			"diagram_failed":    stats.diagramFailed,
			"diagram_fallback":  stats.fallbackSuccess,
			"table_total":       stats.tableTotal,
			"table_success":     stats.tableSuccess,
			"table_failed":      stats.tableFailed,
			"image_skipped":     stats.imageSkipped,
		})
	}

	fmt.Printf("替换成功！\n")
	fmt.Printf("  文档 ID: %s\n", req.documentID)
	fmt.Printf("  父块 ID: %s\n", req.parentBlockID)
	fmt.Printf("  替换前子块数: %d\n", oldCount)
	fmt.Printf("  新增顶层块数: %d\n", topLevelCreated)
	fmt.Printf("  总块数: %d\n", stats.totalBlocks)
	if stats.imageSkipped > 0 {
		fmt.Printf("  图片: %d 张 (已创建空占位块，飞书 API 暂不支持通过 Open API 插入图片)\n", stats.imageSkipped)
	}
	if stats.tableTotal > 0 {
		fmt.Printf("  表格: %d/%d 成功\n", stats.tableSuccess, stats.tableTotal)
	}
	if stats.diagramTotal > 0 {
		if stats.fallbackSuccess > 0 {
			fmt.Printf("  图表: %d/%d 成功 (%d 降级为代码块)\n", stats.diagramSuccess, stats.diagramTotal, stats.fallbackSuccess)
		} else {
			fmt.Printf("  图表: %d/%d 成功\n", stats.diagramSuccess, stats.diagramTotal)
		}
	}
	fmt.Printf("  链接: https://feishu.cn/docx/%s\n", req.documentID)
	return nil
}

func cleanupAppendedChildren(documentID, parentBlockID string, oldCount int) error {
	children, err := client.GetAllBlockChildren(documentID, parentBlockID)
	if err != nil {
		return fmt.Errorf("获取当前子块失败: %w", err)
	}
	if len(children) <= oldCount {
		return nil
	}
	if err := client.DeleteBlocks(documentID, parentBlockID, oldCount, len(children)); err != nil {
		return fmt.Errorf("删除新增子块失败: %w", err)
	}
	return nil
}

func init() {
	docCmd.AddCommand(replaceContentCmd)
	replaceContentCmd.Flags().StringP("content", "c", "", "Markdown 内容")
	replaceContentCmd.Flags().String("content-file", "", "Markdown 文件路径")
	replaceContentCmd.Flags().String("source-type", "", "源类型 (file/content)")
	replaceContentCmd.Flags().StringP("block-id", "b", "", "目标父块 ID (默认: 文档根节点)")
	replaceContentCmd.Flags().Bool("allow-empty", false, "允许空内容，此时会清空目标父块")
	replaceContentCmd.Flags().BoolP("force", "f", false, "跳过确认直接执行")
	replaceContentCmd.Flags().Bool("upload-images", true, "上传 Markdown 中的本地图片")
	replaceContentCmd.Flags().Int("diagram-workers", 5, "图表并发数")
	replaceContentCmd.Flags().Int("table-workers", 3, "表格并发数")
	replaceContentCmd.Flags().Int("diagram-retries", 10, "图表重试次数")
	replaceContentCmd.Flags().StringP("output", "o", "", "输出格式 (json)")
}
