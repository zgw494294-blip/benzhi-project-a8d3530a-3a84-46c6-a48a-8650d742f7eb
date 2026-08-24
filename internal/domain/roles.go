package domain

type Role string

const (
	RoleMonitor    Role = "monitor"
	RoleRemediator Role = "remediator"
	RoleReviewer   Role = "reviewer"
)

func RequireRole(actual Role, allowed ...Role) error {
	for _, role := range allowed {
		if actual == role {
			return nil
		}
	}
	return NewError(CodeForbidden, "角色 %s 无权执行此操作", actual)
}
