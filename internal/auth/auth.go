package auth

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
)

// 定义 context key
type contextKey string

const (
	// UserIDKey 用户ID的context key（uid，字符串 UUID）
	UserIDKey contextKey = "user_id"
	// UserRoleKey 用户角色的context key
	UserRoleKey contextKey = "user_role"
)

// Role 用户角色
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// GetUIDFromContext 从context中获取用户ID（字符串 UUID）
func GetUIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(UserIDKey).(string)
	return uid, ok
}

// GetRoleFromContext 从context中获取用户角色
func GetRoleFromContext(ctx context.Context) (Role, bool) {
	role, ok := ctx.Value(UserRoleKey).(Role)
	return role, ok
}

// IsAdmin 判断当前用户是否为管理员
func IsAdmin(ctx context.Context) bool {
	role, ok := GetRoleFromContext(ctx)
	return ok && role == RoleAdmin
}

// CheckOwnership 检查用户是否有权限访问指定资源
func CheckOwnership(ctx context.Context, resourceUID string) error {
	currentUID, ok := GetUIDFromContext(ctx)
	if !ok {
		return errors.Unauthorized("UNAUTHORIZED", "authentication required")
	}

	// 管理员可以访问所有资源
	if IsAdmin(ctx) {
		return nil
	}

	// 普通用户只能访问自己的资源
	if currentUID != resourceUID {
		return errors.Forbidden("FORBIDDEN", "permission denied: you can only access your own resources")
	}

	return nil
}
