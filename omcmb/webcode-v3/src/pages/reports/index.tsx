import { StubPage } from '@/pages/_stub'

export function ReportsPage() {
  return (
    <StubPage
      code="F06"
      title="REPORTS · 简报中心"
      subtitle="SCHEDULED REPORT MILL"
      plan={[
        '报表定义网格：类型 × 频率 × 状态 × 上次运行',
        '一键预览（HTML/PDF/Excel）+ 在线编辑模板',
        '订阅管理：邮件 / 飞书 / Webhook 推送',
        '历史归档：按月折叠的 timeline + 文件大小柱图',
      ]}
    />
  )
}
