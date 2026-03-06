package cmd

import (
	"github.com/spf13/cobra"
)

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "文档操作命令",
	Long: `文档操作命令，包括创建、获取、编辑、删除文档及块内容。

子命令:
  create      创建新文档
  get         获取文档信息
  blocks      获取文档所有块
  add         向文档添加内容
  replace     用 Markdown 替换父块下的全部子块
  update      更新块内容
  delete      删除块
  export      导出文档为 Markdown
  import      从 Markdown 导入文档
  export-file 导出文档为文件（PDF/DOCX/XLSX）
  import-file 导入文件为云文档

示例:
  # 创建文档
  feishu-cli doc create --title "我的文档"

  # 获取文档信息
  feishu-cli doc get <document_id>

  # 导出为 Markdown
  feishu-cli doc export <document_id> --output doc.md

  # 导出为 PDF
  feishu-cli doc export-file <document_id> --type pdf -o output.pdf

  # 导入 Word 文档
  feishu-cli doc import-file report.docx --type docx`,
}

func init() {
	rootCmd.AddCommand(docCmd)
}
