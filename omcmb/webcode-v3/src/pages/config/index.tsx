import { StubPage } from '@/pages/_stub'

export function ConfigPage() {
  return (
    <StubPage
      code="F02"
      title="CONFIG · 配置编排"
      subtitle="TR-069 PARAMETER TREE · TEMPLATE / BASELINE"
      plan={[
        '左侧：参数树（Device.* / InternetGatewayDevice.*）+ 节点搜索',
        '中央：选中节点的 GET / SET 表单 + 当前值 / 默认值 / 最近变更',
        '右侧：批量下发任务面板 + 状态时间线',
        '上行：模板库 / 基线对比 / 三级回退（产品 → OUI → 运营商默认）',
      ]}
      blocks={[
        { label: 'TEMPLATES', value: '—', tone: 'cyan' },
        { label: 'IN-PROGRESS', value: '—', tone: 'amber' },
        { label: 'SUCCESS 24H', value: '—', tone: 'lime' },
        { label: 'FAILED 24H', value: '—', tone: 'magenta' },
      ]}
    />
  )
}
