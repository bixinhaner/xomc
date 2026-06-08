package mml

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// isUniqueViolation 判断 err 是否为 PostgreSQL unique_violation（SQLSTATE 23505）。
// 用于把 DB 兜底唯一索引（如 mml_custom_command 私有命名空间索引、mml_commands
// command_name/command_code 唯一索引）的并发 race 冲突翻译成业务级 409，
// 避免裸 500 + 泄露约束名。常规重名由 service 层查询预检拦截，本函数只兜 race。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
