// Package response 提供统一的 API 响应信封 helper（v0.1）。
//
// 信封约定（参 docs/architecture/api-envelope.md 决议 D）：
//
//	{
//	  "ret":  1 | 0,           // 1=成功 / 0=失败（注意与历史 code=0 含义相反，按业务团队约定）
//	  "msg":  "ok" | "...",    // 与 ret 同步：成功固定 "ok"；失败为可读错误描述
//	  "data": <any> | null     // 业务数据节点；失败时固定 null
//	}
//
// 错误信封额外可选字段：
//
//	"biz_code":   int     // BusinessError.Code，运维细粒度定位用
//	"request_id": string  // 请求 ID（来自 ctx），便于排障
//
// 设计原则：
//  1. HTTP 状态码语义保留 — 成功 2xx，失败 4xx/5xx；ret 与 status 同步表达
//  2. 流式响应（SSE / 文件下载）不应使用本包，handler 直接写 ResponseWriter
//  3. 现网错误路径统一走 errors.AbortWithError，本包不重复实现 redaction
//
// 迁移指南：
//
//	旧：c.JSON(http.StatusOK, role)
//	新：response.OK(c, role)
//
//	旧：c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "msg": "查询成功"})
//	新：response.OKWithMsg(c, result, "查询成功")
//
//	旧：c.JSON(http.StatusNoContent, nil)
//	新：response.OK(c, nil)   // 注意：HTTP 仍为 200，body 是 {"ret":1,"msg":"ok","data":null}
//	     如必须 204（无 body），仍直接用 c.Status(204)
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MsgOK 是成功响应的默认 msg 字段值。
const MsgOK = "ok"

// OK writes a success response with HTTP 200 and the standard envelope.
// data 可为 nil（用于纯写操作的成功响应）。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"ret":  1,
		"msg":  MsgOK,
		"data": data,
	})
}

// OKWithStatus writes a success response with custom HTTP status (e.g. 201 Created).
func OKWithStatus(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, gin.H{
		"ret":  1,
		"msg":  MsgOK,
		"data": data,
	})
}

// OKWithMsg writes a success response with a custom human-readable message.
// 主要用于把历史 {code:0,data,msg:"创建成功"} 形式映射过来。
func OKWithMsg(c *gin.Context, data any, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"ret":  1,
		"msg":  msg,
		"data": data,
	})
}

// Fail writes a failure response with the given HTTP status and message.
// 业务错误码（biz_code）此处不传；如需保留 BusinessError.Code，使用 errors.AbortWithError
// 或本包的 FailWithBizCode。
func Fail(c *gin.Context, statusCode int, msg string) {
	c.AbortWithStatusJSON(statusCode, gin.H{
		"ret":  0,
		"msg":  msg,
		"data": nil,
	})
}

// FailWithBizCode writes a failure response with a business error code.
// 用于 errors.AbortWithError 内部统一构造 body。
func FailWithBizCode(c *gin.Context, statusCode int, msg string, bizCode int) {
	body := gin.H{
		"ret":  0,
		"msg":  msg,
		"data": nil,
	}
	if bizCode != 0 {
		body["biz_code"] = bizCode
	}
	c.AbortWithStatusJSON(statusCode, body)
}

// FailWithData writes a failure response with a structured data payload.
// 适用于校验失败等需要把详细错误列表回传给前端的场景。
func FailWithData(c *gin.Context, statusCode int, msg string, data any) {
	c.AbortWithStatusJSON(statusCode, gin.H{
		"ret":  0,
		"msg":  msg,
		"data": data,
	})
}
