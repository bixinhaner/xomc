package notification

import (
	"fmt"
	"strings"
	"text/template"
)

// RenderTemplate 用 vars 渲染模板的 Subject 与 Body。
//
// 模板语法为 Go text/template（占位符形如 {{.device_sn}}）。缺失的变量按
// missingkey=zero 渲染为空串，而非报错或留下 "<no value>" —— 通知邮件宁可
// 少一个字段，也不该因模板与变量不同步而整封发不出。
func RenderTemplate(tpl *NotificationTemplate, vars map[string]string) (subject, body string, err error) {
	if tpl == nil {
		return "", "", fmt.Errorf("nil template")
	}
	subject, err = renderText("subject", tpl.Subject, vars)
	if err != nil {
		return "", "", fmt.Errorf("render subject: %w", err)
	}
	body, err = renderText("body", tpl.Body, vars)
	if err != nil {
		return "", "", fmt.Errorf("render body: %w", err)
	}
	return subject, body, nil
}

// renderText 编译并执行单段模板文本。
func renderText(name, text string, vars map[string]string) (string, error) {
	t, err := template.New(name).Option("missingkey=zero").Parse(text)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, vars); err != nil {
		return "", err
	}
	return b.String(), nil
}
