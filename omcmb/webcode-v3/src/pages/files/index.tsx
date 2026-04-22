import { StubPage } from '@/pages/_stub'

export function FilesPage() {
  return (
    <StubPage
      code="F06"
      title="FILES · 档案库"
      subtitle="OBJECT STORAGE EXPLORER · MinIO"
      plan={[
        '桶 / 对象 / 标签 三层视图，用扇形 sunburst 呈现占用',
        '上传 / 下载 / 校验 / 软删 / 永久删 全套操作',
        '文件类型分布饼图 + 上传速率 sparkline',
        '与设备绑定（PM/MR/备份/固件分流）',
      ]}
    />
  )
}
