import { StubPage } from '@/pages/_stub'

export function BackupPage() {
  return (
    <StubPage
      code="F06"
      title="BACKUP · 备援存档"
      subtitle="DEVICE CONFIG SNAPSHOT · RESTORE"
      plan={[
        '备份策略卡：周期 / 范围 / 保留期 / 上传到 MinIO 的桶',
        '正在执行的批量备份任务：分组进度 + 失败设备 SN 列表',
        '历史备份矩阵：设备 × 时间，可点格子查看 diff',
        '一键恢复：选择目标备份 → 多设备并发 RESTORE',
      ]}
    />
  )
}
