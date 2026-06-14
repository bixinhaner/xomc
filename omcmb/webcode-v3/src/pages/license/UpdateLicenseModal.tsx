import { useCallback, useMemo, useRef, useState } from 'react'
import { CloudUpload, FileText, AlertTriangle, CircleCheck } from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'
import { useUpdateSystemLicense } from '@core/hooks/api/useSystemLicense'
import {
  SystemLicenseErrorCodes,
  extractLicenseErrorCode,
} from '@core/services/api/systemLicenseApi'

import { Modal } from './Modal'

/**
 * UpdateLicenseModal — 系统 license 上传向导（v3 HUD 版，等价 v1 UpdateModal）。
 *
 * 流程：选择 / 拖入 .json|.lic → FileReader 读文本 + 本地 JSON.parse 预览（仅展示，不验签）
 *      → 确认 POST /system-license → 后端 strict 验签 + 替换 → 成功/失败按 biz_code 提示。
 */

interface ParsedPreview {
  raw: string
  licenseId?: string
  licenseType?: string
  issuedAt?: string
  expiryDate?: string
  devicesSupport?: Record<string, number>
  hasSignature: boolean
}

function tryParse(raw: string): { ok: true; preview: ParsedPreview } | { ok: false; msg: string } {
  try {
    const obj = JSON.parse(raw)
    if (!obj || typeof obj !== 'object') {
      return { ok: false, msg: '不是合法的 JSON 对象' }
    }
    const o = obj as Record<string, unknown>
    return {
      ok: true,
      preview: {
        raw,
        licenseId: typeof o.license_id === 'string' ? o.license_id : undefined,
        licenseType: typeof o.license_type === 'string' ? o.license_type : undefined,
        issuedAt: typeof o.issued_at === 'string' ? o.issued_at : undefined,
        expiryDate: typeof o.expiry_date === 'string' ? o.expiry_date : undefined,
        devicesSupport:
          o.devices_support && typeof o.devices_support === 'object'
            ? (o.devices_support as Record<string, number>)
            : undefined,
        hasSignature: typeof o.signature === 'string' && o.signature.length > 0,
      },
    }
  } catch (err) {
    return { ok: false, msg: (err as Error).message }
  }
}

export function UpdateLicenseModal({
  open,
  onClose,
  onDone,
}: {
  open: boolean
  onClose: () => void
  /** 成功后给页面回传一条 banner 文案 */
  onDone?: (msg: string) => void
}) {
  const [fileName, setFileName] = useState<string | null>(null)
  const [preview, setPreview] = useState<ParsedPreview | null>(null)
  const [parseError, setParseError] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useUpdateSystemLicense()

  const reset = useCallback(() => {
    setFileName(null)
    setPreview(null)
    setParseError(null)
    setSubmitError(null)
    setDragOver(false)
  }, [])

  const close = useCallback(() => {
    reset()
    onClose()
  }, [reset, onClose])

  const ingest = useCallback((file: File) => {
    setFileName(file.name)
    setSubmitError(null)
    const reader = new FileReader()
    reader.onload = () => {
      const result = tryParse(String(reader.result ?? ''))
      if (result.ok) {
        setPreview(result.preview)
        setParseError(null)
      } else {
        setPreview(null)
        setParseError(result.msg)
      }
    }
    reader.onerror = () => {
      setPreview(null)
      setParseError(reader.error?.message ?? 'FileReader 读取失败')
    }
    reader.readAsText(file)
  }, [])

  const canSubmit = useMemo(
    () => preview !== null && !parseError && !mutation.isPending,
    [preview, parseError, mutation.isPending],
  )

  const submit = () => {
    if (!preview) return
    setSubmitError(null)
    mutation.mutate(preview.raw, {
      onSuccess: (data) => {
        const msg = data.replaced
          ? `授权已更新 · 替换了旧 license ${data.replaced.licenseId}`
          : '授权已写入 · 首次安装成功'
        onDone?.(msg)
        close()
      },
      onError: (err) => {
        const code = extractLicenseErrorCode(err)
        const raw = (err as Error)?.message ?? ''
        switch (code) {
          case SystemLicenseErrorCodes.SignatureVerifyFailed:
            setSubmitError('签名校验失败 · 该 license 文件签名无效或被篡改')
            break
          case SystemLicenseErrorCodes.IDExists:
            setSubmitError('该 license_id 已存在，无法重复导入')
            break
          case SystemLicenseErrorCodes.InvalidFormat:
            setSubmitError('文件格式错误 · 缺少必填字段或 JSON 结构非法')
            break
          default:
            setSubmitError(raw || '上传失败')
        }
      },
    })
  }

  return (
    <Modal
      open={open}
      title="UPDATE LICENSE · 上传系统授权"
      subtitle="DRAG OR SELECT .json / .lic · STRICT SIGNATURE VERIFY"
      onClose={close}
      footer={
        <>
          <NeonButton onClick={close} disabled={mutation.isPending}>
            取消
          </NeonButton>
          <NeonButton onClick={submit} disabled={!canSubmit}>
            {mutation.isPending ? '验签中…' : '确认上传'}
          </NeonButton>
        </>
      }
    >
      {/* 拖拽 / 选择区 */}
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault()
          setDragOver(true)
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragOver(false)
          const f = e.dataTransfer.files?.[0]
          if (f) ingest(f)
        }}
        className={`flex w-full flex-col items-center justify-center gap-2 rounded-sm border border-dashed py-8 transition-colors ${
          dragOver
            ? 'border-cyan-400/80 bg-cyan-500/10'
            : 'border-cyan-500/30 bg-cyan-500/[0.03] hover:border-cyan-400/60'
        }`}
      >
        <CloudUpload className="size-8 text-cyan-300/70" />
        <div className="font-mono text-xs uppercase tracking-[0.18em] text-cyan-300/70">
          {fileName ?? '点击选择或拖入授权文件'}
        </div>
        <div className="font-mono text-[10px] text-cyan-300/45">支持 .json / .lic · 单文件</div>
        <input
          ref={inputRef}
          type="file"
          accept=".json,.lic"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0]
            if (f) ingest(f)
            e.target.value = ''
          }}
        />
      </button>

      {/* 解析错误 */}
      {parseError ? (
        <div className="mt-3 flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-3 py-2 text-rose-300">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          <div className="font-mono text-xs">解析失败 · {parseError}</div>
        </div>
      ) : null}

      {/* 预览 */}
      <div className="mt-4">
        <div className="mb-2 flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/60">
          <FileText className="size-3.5" />
          PREVIEW · 本地预览（不代表验签结果）
        </div>
        {preview ? (
          <div className="grid grid-cols-2 gap-px overflow-hidden rounded-sm border border-cyan-500/15 bg-cyan-500/10">
            <PreviewCell label="LICENSE ID" value={preview.licenseId ?? '—'} mono />
            <PreviewCell label="TYPE" value={preview.licenseType ?? '—'} />
            <PreviewCell label="ISSUED" value={preview.issuedAt ?? '—'} mono />
            <PreviewCell label="EXPIRY" value={preview.expiryDate ?? '永久'} mono />
            <div className="col-span-2 bg-[#070b18] px-3 py-2">
              <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                DEVICES SUPPORT
              </div>
              <div className="mt-1.5 flex flex-wrap gap-1.5">
                {preview.devicesSupport && Object.keys(preview.devicesSupport).length > 0 ? (
                  Object.entries(preview.devicesSupport).map(([k, v]) => (
                    <span
                      key={k}
                      className="rounded-sm border border-cyan-500/40 bg-cyan-500/10 px-2 py-0.5 font-mono text-[11px] text-cyan-200"
                    >
                      {k}: {v}
                    </span>
                  ))
                ) : (
                  <span className="font-mono text-xs text-cyan-300/45">—</span>
                )}
              </div>
            </div>
            <div className="col-span-2 flex items-center gap-2 bg-[#070b18] px-3 py-2">
              <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                SIGNATURE
              </div>
              {preview.hasSignature ? (
                <span className="inline-flex items-center gap-1 font-mono text-xs text-emerald-300">
                  <CircleCheck className="size-3.5" /> 已包含签名
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 font-mono text-xs text-amber-300">
                  <AlertTriangle className="size-3.5" /> 无签名 · strict 模式将被拒绝
                </span>
              )}
            </div>
          </div>
        ) : (
          <div className="rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] px-3 py-6 text-center font-mono text-xs text-cyan-300/45">
            尚未选择文件
          </div>
        )}
      </div>

      {/* 提交错误 */}
      {submitError ? (
        <div className="mt-3 flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-3 py-2 text-rose-300">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          <div className="font-mono text-xs">{submitError}</div>
        </div>
      ) : null}
    </Modal>
  )
}

function PreviewCell({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="bg-[#070b18] px-3 py-2">
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
        {label}
      </div>
      <div className={`mt-0.5 truncate text-sm text-cyan-100 ${mono ? 'font-mono text-xs' : ''}`}>
        {value}
      </div>
    </div>
  )
}
