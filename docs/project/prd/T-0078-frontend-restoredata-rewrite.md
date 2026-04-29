# PRD: 前端 RestoreData wholesale rewrite（T-0078 / T-0072 followup）

> **关联**: Backlog T-0078 / Sprint-07..08 / Domain=frontend / Type=feat
> **作者**: Claude（代 Owner=前端）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: MVP 改为手动路径输入（filemanager 路径架构不匹配 → 文件浏览器拆 T-0081 / T-0079 后做）

---

## 1. 业务背景

T-0072 后端 MVP 已交付：`POST /api/v1/backup/restore` + `GET /backup/restore-tasks` + `GET /backup/restore-tasks/:id`。配套 FE 页面 `omcmb/webcode/src/pages/backup/RestoreData/index.tsx` 当前 1123 行纯 mock（T-0016 audit 发现）— 与 T-0072 endpoint 完全脱节。

本任务 wholesale rewrite RestoreData，接 T-0072 endpoints 真入。

---

## 2. ULTRATHINK 决策

### 2.1 文件浏览器 scope 收紧

T-0078 原描述含"文件浏览器（接 filemanager listing）"。审计发现：

| 数据源 | 内容 | 是否含备份文件 |
|--------|------|--------------|
| `filemanager` (managed_files 表) | 用户上传 / 系统报表的 DB-tracked 文件 | ❌ 不含 — 备份走 TR-069 upload 直落 MinIO |
| `backup_tasks.file_path` | T-0072 PRD §9.x 表字段已存在 | ⚠️ 当前 NULL（T-0079 拆解到时才填） |
| MinIO 原始对象列表 | 后端无 endpoint | ❌ 需新 backend 工作 |

**结论**：文件浏览器无现成数据源。

**MVP 改为手动路径输入**：表单两栏 `{bucket, object_path}` 让运维直接输入。
- 短期：运维可从 MinIO console / 已有 filemanager UI 复制路径
- 中期：T-0079 后填充 `backup_tasks.file_path` → 增加"从备份历史选"下拉
- 长期：可独立任务 T-0081 加 MinIO 原始 listing endpoint

文件浏览器降为非目标（NMVP），不在本任务交付。

### 2.2 多皮肤影响

`omcmb/frontend-core/` 改类型 + Hook + i18n + Mock → 检查 `webcode-v2/` 和 `webcode-v3/` 是否引用 `RestoreData` 或新增的 hooks → 评估影响。

T-0070 BackupSchedule 同模式时确认多皮肤零影响（v2/v3 不引用 backup pages）。本任务大概率同。

### 2.3 测试 / verify 边界

前端测试基础设施（Vitest + Playwright）目前覆盖度有限。本任务 verify 目标：
- `npx tsc --noEmit` ✅ 严格模式
- `npm run lint` 不引入新 problems
- 手工冒烟（dev 模式访问 `/backup/restore-data` 验证 rewrite 不崩、mock 模式可见 mock 数据、real 模式可看到真 endpoint 调用）

不做 unit/E2E 测试 — 与 T-0070 / T-0016 / T-0071 的 FE 交付惯例一致。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我有备份对象路径（如 `config_backup/backup/2026/04/29/cfg.xml.gz`），想恢复到 SN001 + SN002，UI 表单输入 → 创建 restore 任务，看到列表中状态变化 |
| 网管开发者 | i18n key 复用 T-0070/T-0071 的命名（`backup.*`），新增 `backup.restore.*` 子段 |
| QA | 手工冒烟：mock=true 见 mock，mock=false 表单提交进 POST /backup/restore 真 endpoint |

---

## 4. 验收标准（GWT）

### V1 — 列表渲染（mock 模式）
- **Given** `VITE_USE_MOCK=true`
- **When** 访问 `/backup/restore-data`
- **Then** 列表渲染 ≥ 1 条 mock RestoreTask；含 source_object_path / target_count / status badge / progress

### V2 — 列表渲染（真后端）
- **Given** `VITE_USE_MOCK=false` 且后端 `GET /backup/restore-tasks` 返回 `{items: [], total: 0, ...}`
- **When** 访问页面
- **Then** 空状态 UI（"暂无恢复任务"）；不报错

### V3 — 创建 restore（核心）
- **Given** 用户点击"创建恢复"按钮
- **When** Drawer 打开 → 填表 `bucket=config_backup`, `object_path=backup/2026/04/29/cfg.xml.gz`, `target_device_sns=[SN001,SN002]` → 提交
- **Then** 调用 `POST /backup/restore` (real) 或 mock `createRestore` → 成功 toast → Drawer 关闭 → 列表刷新

### V4 — 路径校验（FE 层防御）
- **When** `bucket` 或 `object_path` 含 `..`，或 `bucket != "config_backup"`，或 `target_device_sns` 为空
- **Then** Form validation 阻断提交，显示错误提示（与后端 validateRestorePath 同语义但前置校验，避免不必要的 400 RTT）

### V5 — 状态轮询
- **Given** 列表中有 status=pending / running 的任务
- **When** 页面停留
- **Then** React Query refetchInterval 5s 拉取列表；状态自动更新至 completed/failed

### V6 — 错误显示
- **Given** restore_task 含 error_message（如 `skipped: SN_GHOST`）
- **When** 列表渲染该行
- **Then** error_message 通过 Tooltip 或 Alert 显示

### V7 — i18n
- **Given** 切换 zh-CN / en-US
- **Then** 所有 backup.restore.* 文本正确切换；不出现裸键名

### V8 — TypeScript 严格
- **Given** 全部新增/修改文件
- **When** `cd omcmb/webcode && npm run typecheck`
- **Then** 0 errors；禁止新增 `any`/`interface{}`；BackendXxx → mapBackendXxx → frontend Xxx 模式

### V9 — 多皮肤兼容
- **Given** `cd omcmb/webcode-v2 && npm run typecheck` 和同样 v3
- **When** 不引用 RestoreData 的 v2/v3 编译
- **Then** 0 errors（业务层 frontend-core 类型扩展不破坏 v2/v3 编译）

---

## 5. 运营商差异矩阵

无差异（restore UI 不感知运营商；CPE 端 Download(FileType=3) 由 carrier 适配器在后端处理，前端只展示）。

---

## 6. 非目标

| # | 非目标 | 后续承接 |
|---|--------|---------|
| N1 | 文件浏览器（接 MinIO listing 或 backup_tasks.file_path 选择器） | T-0079 后做（Backup 历史选 picker）；或 T-0081（MinIO listing endpoint） |
| N2 | restore 进度的设备级回写（每设备 device_task 状态聚合） | T-0072 PRD §6 N4 已 defer |
| N3 | restore 历史 detail 页面（点行跳详情） | MVP 不做；表格内 expandable row 替代 |
| N4 | Vitest 单元 / Playwright E2E 测试 | 与既有 FE 任务一致暂不做 |
| N5 | restore 取消功能（需要后端 cancel endpoint） | 后续后端任务 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0072 后端 endpoint | ✅ done |
| `frontend-core/services/api/backupApi.ts` | 已存在；本任务扩展 |
| `frontend-core/hooks/api/useBackup.ts` | 已存在；本任务扩展 |
| `frontend-core/mock/data/backup.ts` | 已存在；本任务扩展 |

无新外部 npm dep。

---

## 8. 度量

- 行数 delta：1123 → ~500（净减 ~600；mock 残留 → 真 endpoint）
- i18n keys 增量：~12 个 zh-CN + 同等 en-US
- typecheck 0 errors（webcode + webcode-v2 + webcode-v3 三皮肤）
- lint baseline 不增

---

## 9. 设计备忘（S2）

### 9.1 frontend-core 扩展

**`mock/data/backup.ts`**：新增 RestoreTask 类型 + 5 条 mock 数据
```typescript
export interface RestoreTask {
  id: string;
  sourceBucket: string;
  sourceObjectPath: string;
  targetDeviceSns: string[];
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number;
  errorMessage?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
}
```

**`services/api/backupApi.ts`**：扩展 BackendRestoreTask + mapBackendRestoreTask + 3 方法
```typescript
interface BackendRestoreTask {
  id: string;
  source_bucket: string;
  source_object_path: string;
  target_device_sns: string[];
  status: string;
  progress: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
}

function mapBackendRestoreTask(b: BackendRestoreTask): RestoreTask { ... }

export const backupApi = {
  // ... existing methods
  async createRestore(req: { bucket: string; object_path: string; target_device_sns: string[] }): Promise<RestoreTask>,
  async listRestoreTasks(params: PageRequest): Promise<PageResponse<RestoreTask>>,
  async getRestoreTask(id: string): Promise<RestoreTask>,
};
```

**`hooks/api/useBackup.ts`**：3 个新 hook
```typescript
export function useBackupRestoreTasks(params: PageRequest) {
  return useQuery({
    queryKey: ['backup', 'restore-tasks', params],
    queryFn: () => useMock ? backupService.listRestoreTasks(params) : backupApi.listRestoreTasks(params),
    refetchInterval: 5000, // poll while in-flight; could refine to "only if has pending/running"
  });
}

export function useBackupRestoreTask(id: string | undefined) { ... }

export function useCreateBackupRestore() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req) => useMock ? ... : backupApi.createRestore(req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['backup', 'restore-tasks'] }),
  });
}
```

**`i18n/zh-CN.ts` / `en-US.ts`**：新增 `backup.restore.*` 段
- `backup.restore.title`
- `backup.restore.create`
- `backup.restore.bucket`
- `backup.restore.objectPath`
- `backup.restore.targetDevices`
- `backup.restore.submit`
- `backup.restore.empty`
- `backup.restore.statusPending` / Running / Completed / Failed / Cancelled
- `backup.restore.errorTooltip`
- `backup.restore.bucketHint`
- `backup.restore.pathHint`

### 9.2 webcode 重写

**`pages/backup/RestoreData/index.tsx`**：~400-500 行，结构：

```typescript
function RestoreData() {
  const { t } = useT();
  const [params, setParams] = useState<PageRequest>({ page: 1, pageSize: 20 });
  const { data, isLoading } = useBackupRestoreTasks(params);
  const createMutation = useCreateBackupRestore();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [form] = Form.useForm();

  const columns: ColumnsType<RestoreTask> = useMemo(() => [
    { title: 'ID', dataIndex: 'id', width: 220, ellipsis: true },
    { title: t('backup.restore.sourcePath'), render: (_, r) => `${r.sourceBucket}/${r.sourceObjectPath}` },
    { title: t('backup.restore.targetCount'), render: (_, r) => r.targetDeviceSns.length },
    { title: t('common.status'), render: (_, r) => <StatusBadge status={r.status} /> },
    { title: t('common.progress'), render: (_, r) => <Progress percent={r.progress} size="small" /> },
    { title: t('common.error'), render: (_, r) => r.errorMessage ? <Tooltip title={r.errorMessage}><Tag color="error">!</Tag></Tooltip> : null },
    { title: t('common.createdAt'), dataIndex: 'createdAt' },
  ], [t]);

  const handleSubmit = async () => {
    const values = await form.validateFields();
    await createMutation.mutateAsync({
      bucket: values.bucket,
      object_path: values.objectPath,
      target_device_sns: values.targetDeviceSns.split('\n').filter(Boolean),
    });
    setDrawerOpen(false);
    form.resetFields();
  };

  return (
    <ListPageLayout
      title={t('backup.restore.title')}
      extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawerOpen(true)}>{t('backup.restore.create')}</Button>}
    >
      <DataTable columns={columns} dataSource={data?.items ?? []} loading={isLoading} pagination={...} />
      <Drawer title={t('backup.restore.create')} open={drawerOpen} onClose={() => setDrawerOpen(false)}>
        <Form form={form}>
          <Form.Item name="bucket" label={t('backup.restore.bucket')} rules={[
            { required: true },
            { pattern: /^config_backup$/, message: t('backup.restore.bucketHint') },
          ]}>
            <Input defaultValue="config_backup" />
          </Form.Item>
          <Form.Item name="objectPath" label={t('backup.restore.objectPath')} rules={[
            { required: true },
            { validator: (_, v) => v.includes('..') ? Promise.reject(t('backup.restore.pathTraversal')) : Promise.resolve() },
          ]}>
            <Input placeholder="backup/YYYY/MM/DD/cfg.xml.gz" />
          </Form.Item>
          <Form.Item name="targetDeviceSns" label={t('backup.restore.targetDevices')} rules={[{ required: true }]}>
            <Input.TextArea rows={5} placeholder="SN001\nSN002" />
          </Form.Item>
          <Button type="primary" onClick={handleSubmit} loading={createMutation.isPending}>{t('backup.restore.submit')}</Button>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
```

**StatusBadge** 复用 T-0070/T-0071 模式（不另外抽组件）

### 9.3 待定点

| 待定 | 决策 |
|------|------|
| target_device_sns 输入用 textarea-line-split 还是 `Select mode="tags"`？ | textarea — 与 BackupTasks/BackupSchedule 已建立的"运维粘贴 SN 列表"用法一致；后续可换 Select 但不在 MVP |
| 路径预填 default `config_backup/`？ | bucket 字段 default `config_backup`；object_path placeholder 提示 |
| refetchInterval 5s 全表拉？ | MVP 直接 5s；后续可做"只在有 pending/running 时轮询"优化 |

### 9.4 文件清单

修改：
- `omcmb/frontend-core/src/mock/data/backup.ts` — 加 RestoreTask 类型 + mock 数据
- `omcmb/frontend-core/src/services/api/backupApi.ts` — 加 BackendRestoreTask + mapBackendRestoreTask + createRestore/listRestoreTasks/getRestoreTask
- `omcmb/frontend-core/src/hooks/api/useBackup.ts` — 加 3 个 hook
- `omcmb/frontend-core/src/i18n/zh-CN.ts` — 加 backup.restore.* 段
- `omcmb/frontend-core/src/i18n/en-US.ts` — 同上 EN
- `omcmb/webcode/src/pages/backup/RestoreData/index.tsx` — wholesale rewrite

无新增（除上面修改文件）。
