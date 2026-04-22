import { StubPage } from '@/pages/_stub'

export function LicensePage() {
  return (
    <StubPage
      code="F06"
      title="LICENSE · 印玺管理"
      subtitle="CAPACITY · EXPIRY · UPGRADE"
      plan={[
        '容量仪表盘：已用 / 已购 / 剩余（用大圆环 + 倒计时）',
        '到期日历 + 提前 30/60/90 天预警闪烁',
        '功能特性矩阵：勾选状态 + 灰度开关',
        '凭证文件签名校验 + 一键刷新',
      ]}
      blocks={[
        { label: 'USED', value: '0 / 100K', tone: 'lime' },
        { label: 'EXPIRES', value: '— DAYS', tone: 'amber' },
        { label: 'FEATURES', value: '0', tone: 'cyan' },
        { label: 'STATUS', value: 'OK', tone: 'lime' },
      ]}
    />
  )
}
