import { useEffect, useState } from 'react'
import { RotateCcw, Save } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useBackupPolicy,
  useUpdateBackupPolicy,
  useFTPConfigs,
} from '@core/hooks/api/useBackup'
import { DEFAULT_BACKUP_POLICY } from '@core/mock/data/backup'
import type { BackupPolicy as BackupPolicyModel } from '@core/mock/data/backup'

// ============================================================
// 备份策略 — 对齐 v1 webcode/src/pages/backup/BackupPolicy（路由 /backup/policy）
//   · 单例策略：保留 / 自动清理 / 压缩 / 存储 / 加密 / 告警
//   · 清理 / 压缩 / 加密(非 GCM) / 告警 字段持久化但 executor 尚未全部生效 → "尚未生效"标记
// 全部数据走 @core hooks（真实后端，404 优雅回退默认值），三态完整。
// ============================================================

function getErrMsg(e: unknown): string {
  return e instanceof Error ? e.message : '操作失败'
}

function Toggle({
  checked,
  onChange,
}: {
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <button type="button" onClick={() => onChange(!checked)}>
      <Badge variant={checked ? 'success' : 'muted'}>
        {checked ? '开' : '关'}
      </Badge>
    </button>
  )
}

function NotEnforcedTag({ tone = 'info' }: { tone?: 'info' | 'warning' }) {
  return (
    <Badge variant={tone === 'warning' ? 'warning' : 'outline'} className="ml-2">
      尚未生效
    </Badge>
  )
}

function Section({
  title,
  tag,
  children,
}: {
  title: string
  tag?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center text-base">
          {title}
          {tag}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-2">{children}</div>
      </CardContent>
    </Card>
  )
}

function Field({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}

export default function BackupPolicy() {
  const { data: policy, isLoading, isError, error, isFetching, refetch } =
    useBackupPolicy()
  const { data: ftpData } = useFTPConfigs({ page: 1, pageSize: 100 })
  const updatePolicy = useUpdateBackupPolicy()

  const [form, setForm] = useState<BackupPolicyModel>(DEFAULT_BACKUP_POLICY)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(null)

  useEffect(() => {
    if (policy) setForm(policy)
  }, [policy])

  const ftpOptions = ftpData?.items ?? []

  const flash = (kind: 'ok' | 'err', msg: string) => {
    setToast({ kind, msg })
    window.setTimeout(() => setToast(null), 3000)
  }

  const set = <K extends keyof BackupPolicyModel>(
    key: K,
    value: BackupPolicyModel[K]
  ) => setForm((f) => ({ ...f, [key]: value }))

  const handleSave = () => {
    updatePolicy.mutate(form, {
      onSuccess: () => flash('ok', '备份策略已保存'),
      onError: (e: unknown) => flash('err', `保存失败：${getErrMsg(e)}`),
    })
  }

  const handleReset = () => {
    setForm(DEFAULT_BACKUP_POLICY)
    flash('ok', '已恢复默认值（未保存）')
  }

  const encryptionTag = !form.enableEncryption
    ? null
    : form.encryptionAlgorithm === 'AES-256-GCM'
      ? <Badge variant="success" className="ml-2">已生效</Badge>
      : <NotEnforcedTag tone="warning" />

  return (
    <PageShell
      title="备份策略"
      description="全局备份保留、清理、压缩、存储、加密与告警策略"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <Button variant="outline" size="sm" onClick={handleReset}>
            <RotateCcw className="size-4" /> 恢复默认
          </Button>
          <Button
            size="sm"
            className="ml-auto"
            disabled={updatePolicy.isPending || isLoading}
            onClick={handleSave}
          >
            <Save className="size-4" /> {updatePolicy.isPending ? '保存中…' : '保存'}
          </Button>
        </div>
      }
    >
      {toast ? (
        <div
          className={cn(
            'mb-3 rounded-md px-3 py-2 text-sm',
            toast.kind === 'ok'
              ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
              : 'bg-destructive/10 text-destructive'
          )}
        >
          {toast.msg}
        </div>
      ) : null}

      {isError ? (
        <div className="mb-3 flex items-center gap-3 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
          加载策略失败：{error instanceof Error ? error.message : '未知错误'}
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            重试
          </Button>
        </div>
      ) : null}

      {isLoading ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            加载中…
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {/* 保留 */}
          <Section title="保留策略">
            <Field label="保留天数">
              <Select
                value={String(form.retentionDays)}
                onValueChange={(v) => set('retentionDays', Number(v))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[7, 14, 30, 60, 90, 180, 365].map((d) => (
                    <SelectItem key={d} value={String(d)}>
                      {d} 天
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field label="最大备份数">
              <Input
                type="number"
                min={1}
                max={10000}
                value={form.maxBackupCount}
                onChange={(e) => set('maxBackupCount', Number(e.target.value) || 0)}
              />
            </Field>
            <Field label="最小备份数">
              <Input
                type="number"
                min={1}
                max={100}
                value={form.minBackupCount}
                onChange={(e) => set('minBackupCount', Number(e.target.value) || 0)}
              />
            </Field>
          </Section>

          {/* 自动清理 */}
          <Section title="自动清理" tag={<NotEnforcedTag />}>
            <Field label="启用自动清理">
              <div>
                <Toggle
                  checked={form.autoCleanup}
                  onChange={(v) => set('autoCleanup', v)}
                />
              </div>
            </Field>
            {form.autoCleanup ? (
              <>
                <Field label="清理时间">
                  <Select
                    value={form.cleanupTime}
                    onValueChange={(v) => set('cleanupTime', v)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {['00:00', '01:00', '02:00', '03:00', '04:00', '05:00'].map(
                        (v) => (
                          <SelectItem key={v} value={v}>
                            {v}
                          </SelectItem>
                        )
                      )}
                    </SelectContent>
                  </Select>
                </Field>
                <Field label="保留最近 N 份">
                  <Input
                    type="number"
                    min={1}
                    max={50}
                    value={form.keepLastN}
                    onChange={(e) => set('keepLastN', Number(e.target.value) || 0)}
                  />
                </Field>
              </>
            ) : null}
          </Section>

          {/* 压缩 */}
          <Section title="压缩" tag={<NotEnforcedTag />}>
            <Field label="启用压缩">
              <div>
                <Toggle
                  checked={form.enableCompression}
                  onChange={(v) => set('enableCompression', v)}
                />
              </div>
            </Field>
            {form.enableCompression ? (
              <>
                <Field label="压缩格式">
                  <Select
                    value={form.compressionFormat}
                    onValueChange={(v) =>
                      set('compressionFormat', v as BackupPolicyModel['compressionFormat'])
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="gzip">gzip（标准）</SelectItem>
                      <SelectItem value="bzip2">bzip2（高压缩）</SelectItem>
                      <SelectItem value="lz4">lz4（快速）</SelectItem>
                      <SelectItem value="zstd">zstd（均衡）</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>
                <Field label="压缩级别（1-9）">
                  <Input
                    type="number"
                    min={1}
                    max={9}
                    value={form.compressionLevel}
                    onChange={(e) =>
                      set('compressionLevel', Number(e.target.value) || 1)
                    }
                  />
                </Field>
              </>
            ) : null}
          </Section>

          {/* 存储 */}
          <Section title="存储后端">
            <Field label="存储后端">
              <Select
                value={form.storageBackend}
                onValueChange={(v) =>
                  set('storageBackend', v as BackupPolicyModel['storageBackend'])
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="local">本地存储</SelectItem>
                  <SelectItem value="ftp">FTP 服务器</SelectItem>
                  <SelectItem value="sftp">SFTP 服务器</SelectItem>
                  <SelectItem value="nfs">NFS 共享</SelectItem>
                </SelectContent>
              </Select>
            </Field>
            {form.storageBackend === 'local' ? (
              <Field label="本地路径">
                <Input
                  className="font-mono"
                  value={form.localPath}
                  onChange={(e) => set('localPath', e.target.value)}
                  placeholder="/var/backup/omc"
                />
              </Field>
            ) : (
              <Field label="FTP 配置">
                <Select
                  value={form.ftpConfigId ?? ''}
                  onValueChange={(v) => set('ftpConfigId', v)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择 FTP 配置" />
                  </SelectTrigger>
                  <SelectContent>
                    {ftpOptions.length === 0 ? (
                      <SelectItem value="__none__" disabled>
                        暂无可用 FTP 配置
                      </SelectItem>
                    ) : (
                      ftpOptions.map((cfg) => (
                        <SelectItem key={cfg.id} value={cfg.id}>
                          {cfg.configName}（{cfg.host}:{cfg.port}）
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
              </Field>
            )}
            <Field label="最大存储（GB）">
              <Input
                type="number"
                min={1}
                max={100000}
                value={form.maxStorageGB}
                onChange={(e) => set('maxStorageGB', Number(e.target.value) || 0)}
              />
            </Field>
            <Field label="告警阈值（%）">
              <Input
                type="number"
                min={50}
                max={95}
                value={form.alertThresholdPercent}
                onChange={(e) =>
                  set('alertThresholdPercent', Number(e.target.value) || 0)
                }
              />
            </Field>
          </Section>

          {/* 加密 */}
          <Section title="加密" tag={encryptionTag}>
            <Field label="启用加密">
              <div>
                <Toggle
                  checked={form.enableEncryption}
                  onChange={(v) => set('enableEncryption', v)}
                />
              </div>
            </Field>
            {form.enableEncryption ? (
              <Field label="加密算法">
                <Select
                  value={form.encryptionAlgorithm}
                  onValueChange={(v) => set('encryptionAlgorithm', v)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="AES-256-GCM">AES-256-GCM</SelectItem>
                    <SelectItem value="AES-256-CBC">AES-256-CBC</SelectItem>
                    <SelectItem value="ChaCha20-Poly1305">
                      ChaCha20-Poly1305
                    </SelectItem>
                  </SelectContent>
                </Select>
              </Field>
            ) : null}
          </Section>

          {/* 告警 */}
          <Section title="告警" tag={<NotEnforcedTag />}>
            <Field label="失败时告警">
              <div>
                <Toggle
                  checked={form.alertOnFailure}
                  onChange={(v) => set('alertOnFailure', v)}
                />
              </div>
            </Field>
            <Field label="告警邮箱">
              <Input
                type="email"
                value={form.alertEmail}
                onChange={(e) => set('alertEmail', e.target.value)}
                placeholder="admin@example.com"
              />
            </Field>
            <Field label="告警级别">
              <Select
                value={form.alertSeverity}
                onValueChange={(v) =>
                  set('alertSeverity', v as BackupPolicyModel['alertSeverity'])
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="warning">警告</SelectItem>
                  <SelectItem value="major">重要</SelectItem>
                  <SelectItem value="critical">紧急</SelectItem>
                </SelectContent>
              </Select>
            </Field>
          </Section>
        </div>
      )}
    </PageShell>
  )
}
