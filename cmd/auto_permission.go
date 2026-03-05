package cmd

import (
	"fmt"
	"strings"

	"github.com/riba2534/feishu-cli/internal/client"
	"github.com/riba2534/feishu-cli/internal/config"
)

// applyAutoPermissionIfEnabled 在文档创建后按配置自动授权
func applyAutoPermissionIfEnabled(documentID string, docType string) error {
	permissionCfg := config.Get().Permission
	if !permissionCfg.AutoGrant {
		return nil
	}

	memberType := strings.TrimSpace(permissionCfg.MemberType)
	memberID := strings.TrimSpace(permissionCfg.MemberID)
	perm := strings.TrimSpace(permissionCfg.Perm)
	if perm == "" {
		perm = "full_access"
	}

	if memberType == "" {
		return fmt.Errorf("自动授权已开启，但 permission.member_type 为空")
	}
	if memberID == "" {
		return fmt.Errorf("自动授权已开启，但 permission.member_id 为空")
	}
	if perm != "view" && perm != "edit" && perm != "full_access" {
		return fmt.Errorf("自动授权配置无效：permission.perm 仅支持 view/edit/full_access")
	}

	member := client.PermissionMember{
		MemberType: memberType,
		MemberID:   memberID,
		Perm:       perm,
	}
	if err := client.AddPermission(documentID, docType, member, permissionCfg.Notification); err != nil {
		return fmt.Errorf("自动授权失败: %w", err)
	}

	fmt.Printf("已自动授权: %s (%s) -> %s\n", memberID, memberType, perm)
	return nil
}
