import { StubPage } from '@/pages/_stub'

export function SystemPage() {
  return (
    <StubPage
      code="F06"
      title="SYSTEM · 内核管理"
      subtitle="USER · ROLE · AUDIT · TENANT"
      plan={[
        '人员清单：账号 / 角色 / 上次登录 / 在线状态',
        'RBAC 权限矩阵编辑器（角色 × 资源 × 操作）',
        '审计日志检索：按操作人 / 资源 / 时间窗回放',
        '租户切换 / 系统参数（会话超时、密码策略、TLS）',
      ]}
    />
  )
}
