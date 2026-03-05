package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理命令",
	Long: `配置管理命令，用于初始化和管理 CLI 配置。

子命令:
  init    初始化配置文件

配置文件位置:
  ~/.feishu-cli/config.yaml

	配置优先级:
	  环境变量 > 配置文件 > 默认值

	环境变量:
	  FEISHU_APP_ID      应用 ID
	  FEISHU_APP_SECRET  应用密钥
	  FEISHU_PERMISSION_AUTO_GRANT      是否开启创建后自动授权（true/false）
	  FEISHU_PERMISSION_MEMBER_TYPE     成员类型（email/openid/userid/...）
	  FEISHU_PERMISSION_MEMBER_ID       成员标识（被授权用户）
	  FEISHU_PERMISSION_PERM            授权类型（view/edit/full_access）
	  FEISHU_PERMISSION_NOTIFICATION    是否发送授权通知
	  FEISHU_BASE_URL    API 地址（可选）
	  FEISHU_DEBUG       调试模式（可选）

	示例:
	  # 初始化配置文件
	  feishu-cli config init

	  # 使用环境变量
	  export FEISHU_APP_ID="cli_xxx"
	  export FEISHU_APP_SECRET="xxx"
	  export FEISHU_PERMISSION_AUTO_GRANT=true
	  export FEISHU_PERMISSION_MEMBER_TYPE="email"
	  export FEISHU_PERMISSION_MEMBER_ID="user@example.com"
	  export FEISHU_PERMISSION_PERM="full_access"
	  export FEISHU_PERMISSION_NOTIFICATION=true`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
