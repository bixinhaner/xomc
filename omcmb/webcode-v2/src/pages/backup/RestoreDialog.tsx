import { useState } from 'react'
import { X } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'

import {
  useCreateBackupRestore,
  useCreateBackupRestoreByTaskID,
} from '@core/hooks/api/useBackup'

type RestoreMode = 'path' | 'task'

function parseSnList(raw: string): string[] {
  return raw
    .split(/[\r\n,;\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

// 模块内对话框：v2 皮肤暂无共享 Dialog 组件，这里用 Tailwind 叠层自建一个轻量弹窗。
// 业务对齐 v1 RestoreData：支持「按对象路径」与「按备份任务 ID」两种还原来源，
// 目标设备 SN 通过多行/分隔文本批量录入。
export function RestoreDialog({
  open,
  onClose,
  onSuccess,
}: {
  open: boolean
  onClose: () => void
  onSuccess: (msg: string) => void
}) {
  const [mode, setMode] = useState<RestoreMode>('path')
  const [bucket, setBucket] = useState('config_backup')
  const [objectPath, setObjectPath] = useState('')
  const [backupTaskId, setBackupTaskId] = useState('')
  const [snText, setSnText] = useState('')
  const [errMsg, setErrMsg] = useState('')

  const createRestore = useCreateBackupRestore()
  const createByTaskId = useCreateBackupRestoreByTaskID()
  const submitting = createRestore.isPending || createByTaskId.isPending

  if (!open) return null

  const sns = parseSnList(snText)

  const pathInvalid =
    mode === 'path' &&
    (objectPath.trim().length === 0 ||
      objectPath.includes('..') ||
      objectPath.startsWith('/'))
  const taskInvalid = mode === 'task' && backupTaskId.trim().length === 0
  const canSubmit = sns.length > 0 && !pathInvalid && !taskInvalid && !submitting

  const reset = () => {
    setMode('path')
    setBucket('config_backup')
    setObjectPath('')
    setBackupTaskId('')
    setSnText('')
    setErrMsg('')
  }

  const handleClose = () => {
    reset()
    onClose()
  }

  const handleSubmit = () => {
    setErrMsg('')
    const onErr = (e: unknown) =>
      setErrMsg(e instanceof Error ? e.message : '提交失败')
    const onOk = () => {
      onSuccess(`已提交还原任务（目标 ${sns.length} 台设备）`)
      reset()
      onClose()
    }
    if (mode === 'task') {
      createByTaskId.mutate(
        { backupTaskId: backupTaskId.trim(), targetDeviceSns: sns },
        { onSuccess: onOk, onError: onErr }
      )
    } else {
      createRestore.mutate(
        { bucket: bucket.trim(), objectPath: objectPath.trim(), targetDeviceSns: sns },
        { onSuccess: onOk, onError: onErr }
      )
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* 遮罩 */}
      <div
        className="absolute inset-0 bg-black/40"
        onClick={handleClose}
        aria-hidden
      />
      {/* 面板 */}
      <div className="relative z-10 w-full max-w-lg rounded-xl border bg-card shadow-lg">
        <div className="flex items-center justify-between border-b px-5 py-3">
          <h2 className="text-base font-semibold">创建配置还原任务</h2>
          <Button variant="ghost" size="sm" onClick={handleClose}>
            <X className="size-4" />
          </Button>
        </div>

        <div className="space-y-4 px-5 py-4">
          {/* 还原来源 */}
          <div className="space-y-1.5">
            <Label>还原来源</Label>
            <div className="flex gap-2">
              {(['path', 'task'] as const).map((m) => (
                <Button
                  key={m}
                  type="button"
                  size="sm"
                  variant={mode === m ? 'default' : 'outline'}
                  onClick={() => {
                    setMode(m)
                    setErrMsg('')
                  }}
                >
                  {m === 'path' ? '按对象路径' : '按备份任务'}
                </Button>
              ))}
            </div>
          </div>

          {mode === 'path' ? (
            <>
              <div className="space-y-1.5">
                <Label htmlFor="restore-bucket">存储桶</Label>
                <Select value={bucket} onValueChange={setBucket}>
                  <SelectTrigger id="restore-bucket">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="config_backup">config_backup</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="restore-objpath">对象路径</Label>
                <Input
                  id="restore-objpath"
                  value={objectPath}
                  onChange={(e) => setObjectPath(e.target.value)}
                  placeholder="backup/2026/04/29/cfg-xxx.xml.gz"
                />
                {mode === 'path' &&
                  objectPath.trim().length > 0 &&
                  (objectPath.includes('..') || objectPath.startsWith('/')) && (
                    <p className="text-xs text-destructive">
                      路径不可包含 “..” 或以 “/” 开头
                    </p>
                  )}
              </div>
            </>
          ) : (
            <div className="space-y-1.5">
              <Label htmlFor="restore-taskid">备份任务 ID</Label>
              <Input
                id="restore-taskid"
                value={backupTaskId}
                onChange={(e) => setBackupTaskId(e.target.value)}
                placeholder="bkp-xxxx"
              />
              <p className="text-xs text-muted-foreground">
                从备份任务列表复制任务 ID，按其备份产物还原。
              </p>
            </div>
          )}

          <div className="space-y-1.5">
            <Label htmlFor="restore-sns">目标设备 SN</Label>
            <textarea
              id="restore-sns"
              value={snText}
              onChange={(e) => setSnText(e.target.value)}
              rows={5}
              placeholder="每行一个 SN，或用逗号/空格分隔"
              className={cn(
                'flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'
              )}
            />
            <p className="text-xs text-muted-foreground">
              已识别 <span className="font-medium text-foreground">{sns.length}</span> 个 SN
            </p>
          </div>

          {errMsg ? (
            <p className="rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">
              {errMsg}
            </p>
          ) : null}
        </div>

        <div className="flex justify-end gap-2 border-t px-5 py-3">
          <Button variant="outline" size="sm" onClick={handleClose}>
            取消
          </Button>
          <Button size="sm" disabled={!canSubmit} onClick={handleSubmit}>
            {submitting ? '提交中…' : '提交还原'}
          </Button>
        </div>
      </div>
    </div>
  )
}
