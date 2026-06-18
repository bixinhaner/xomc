import { useEffect, useMemo, useState } from 'react'
import { Loader2, Save } from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'
import {
  STANDARD_DATA_TYPES,
  STANDARD_CHANGE_APPLIES,
  dataTypeRangeKind,
  isUnsignedDataType,
} from '@core/types/paramModel'
import type { StandardParam, UpsertStandardInput } from '@core/types/paramModel'

import { Modal } from './Modal'

// ============================================================
// ISSUE-488 (v3 · STARFORGE HUD) — 标准参数树 新增 / 编辑弹窗。
// 业务深度对齐 v1（webcode/src/pages/product/standard-params/index.tsx）
// 与 v2（webcode-v2/src/pages/product/StandardParamDialog.tsx）：
//   字段 standardPath / entryType / access / dataType / changeApplies / min / max
//   - dataType / changeApplies 为下拉枚举（复用 frontend-core 共享常量），非自由文本框
//   - min/max label 随 dataType 动态（string→长度 / 数值→值），仅整数（留空=不校验）
//   - 编辑态历史值兼容：非枚举旧值并入下拉，不静默清空 / 改写
//   - 编辑态 standardPath 禁改
// 设计语言：玻璃拟态霓虹 HUD，复用本模块 Modal + FormRow + neon-input。
// ============================================================

// InputNumber precision=0 等价：只接受整数，空串=不校验。
// 小数按截断（truncate）取整数段，而非删除小数点（避免 12.7→"127" 膨胀）。
// 保留前导负号（signed int 的负 min/max 是合法语义），符号只拼一次（避免 "--5"）。
function toIntString(raw: string): string {
  const trimmed = raw.trim()
  if (trimmed === '') return ''
  const negative = trimmed.startsWith('-')
  const intPart = trimmed.replace(/^-/, '').split('.')[0].replace(/[^\d]/g, '')
  if (intPart === '') return ''
  return String(Number((negative ? '-' : '') + intPart))
}

function FormRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/65">
        {label}
      </div>
      {children}
    </div>
  )
}

export function StandardParamDialog({
  open,
  editing,
  loading,
  error,
  onSubmit,
  onCancel,
}: {
  open: boolean
  editing: StandardParam | null
  loading?: boolean
  error?: boolean
  onSubmit: (value: UpsertStandardInput, path?: string) => void
  onCancel: () => void
}) {
  const [standardPath, setStandardPath] = useState('')
  const [entryType, setEntryType] = useState('parameter')
  const [access, setAccess] = useState('readWrite')
  const [dataType, setDataType] = useState<string>('string')
  const [changeApplies, setChangeApplies] = useState<string>('OnReboot')
  const [minValue, setMinValue] = useState('')
  const [maxValue, setMaxValue] = useState('')
  const [pathErr, setPathErr] = useState(false)

  useEffect(() => {
    if (!open) return
    setPathErr(false)
    if (editing) {
      setStandardPath(editing.standardPath)
      setEntryType(editing.entryType || 'parameter')
      setAccess(editing.access || 'readWrite')
      setDataType(editing.dataType || 'string')
      setChangeApplies(editing.changeApplies || 'OnReboot')
      setMinValue(editing.minValue ?? '')
      setMaxValue(editing.maxValue ?? '')
    } else {
      setStandardPath('')
      setEntryType('parameter')
      setAccess('readWrite')
      setDataType('string')
      setChangeApplies('OnReboot')
      setMinValue('')
      setMaxValue('')
    }
  }, [open, editing])

  // dataType 下拉：枚举集 + 当前历史遗留值（如 STRING / U_INT）并入，不静默清空。
  const dataTypeOptions = useMemo(() => {
    const opts: string[] = [...STANDARD_DATA_TYPES]
    if (dataType && !opts.includes(dataType)) opts.push(dataType)
    return opts
  }, [dataType])

  // changeApplies 下拉：枚举集（带中文标签）+ 历史遗留值（如 reload / immediate 小写）并入。
  const changeAppliesOptions = useMemo(() => {
    const opts: string[] = [...STANDARD_CHANGE_APPLIES]
    if (changeApplies && !opts.includes(changeApplies)) opts.push(changeApplies)
    return opts
  }, [changeApplies])

  // min/max 按 dataType 分三档：string→长度，int/unsignedInt→数值，boolean/dateTime→无范围（禁用）。
  const rangeKind = dataTypeRangeKind(dataType)
  const rangeNone = rangeKind === 'none'
  const minLabel = rangeKind === 'length' ? '最小长度' : '最小值'
  const maxLabel = rangeKind === 'length' ? '最大长度' : '最大值'
  const minBound = isUnsignedDataType(dataType) ? 0 : undefined
  const intPlaceholder = rangeNone ? '该类型无取值范围' : '整数，可空'

  const submit = () => {
    if (loading) return
    if (standardPath.trim() === '') {
      setPathErr(true)
      return
    }
    const norm = (v: string): string | undefined => (v.trim() === '' ? undefined : v.trim())
    const value: UpsertStandardInput = {
      standardPath: standardPath.trim(),
      entryType,
      access,
      dataType,
      changeApplies,
      minValue: norm(minValue),
      maxValue: norm(maxValue),
    }
    onSubmit(value, editing?.standardPath)
  }

  return (
    <Modal
      open={open}
      title={editing ? '编辑标准参数' : '新增标准参数'}
      subtitle={editing ? 'EDIT STANDARD PARAM' : 'NEW STANDARD PARAM'}
      width={560}
      onClose={onCancel}
      footer={
        <>
          <NeonButton onClick={onCancel} disabled={loading}>
            取消
          </NeonButton>
          <NeonButton
            icon={loading ? <Loader2 className="animate-spin" /> : <Save />}
            disabled={loading}
            onClick={submit}
          >
            {editing ? '保存' : '创建'}
          </NeonButton>
        </>
      }
    >
      <div className="space-y-4">
        {error ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-[11px] text-rose-300">
            {editing ? '保存失败，请重试' : '创建失败，请重试'}
          </div>
        ) : null}

        <FormRow label="标准路径 · STANDARD PATH">
          <input
            className="neon-input w-full"
            value={standardPath}
            placeholder="例如 InternetGatewayDevice.DeviceInfo.SoftwareVersion"
            disabled={Boolean(editing)}
            autoFocus={!editing}
            onChange={(e) => {
              setStandardPath(e.target.value)
              if (e.target.value.trim() !== '') setPathErr(false)
            }}
          />
          {pathErr ? (
            <p className="font-mono text-[11px] text-rose-300">标准路径必填</p>
          ) : null}
        </FormRow>

        <div className="grid grid-cols-2 gap-4">
          <FormRow label="条目类型 · ENTRY">
            <select
              className="neon-input w-full cursor-pointer"
              aria-label="条目类型"
              value={entryType}
              onChange={(e) => setEntryType(e.target.value)}
            >
              <option value="parameter">parameter</option>
              <option value="object">object</option>
            </select>
          </FormRow>

          <FormRow label="访问 · ACCESS">
            <select
              className="neon-input w-full cursor-pointer"
              aria-label="访问"
              value={access}
              onChange={(e) => setAccess(e.target.value)}
            >
              <option value="readWrite">readWrite</option>
              <option value="readOnly">readOnly</option>
            </select>
          </FormRow>

          <FormRow label="数据类型 · TYPE">
            <select
              className="neon-input w-full cursor-pointer"
              aria-label="数据类型"
              value={dataType}
              onChange={(e) => {
                const v = e.target.value
                setDataType(v)
                // 切到无取值范围类型（boolean/dateTime）时清空 min/max。
                if (dataTypeRangeKind(v) === 'none') {
                  setMinValue('')
                  setMaxValue('')
                }
              }}
            >
              {dataTypeOptions.map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
          </FormRow>

          <FormRow label="生效方式 · CHANGE APPLIES">
            <select
              className="neon-input w-full cursor-pointer"
              aria-label="生效方式"
              value={changeApplies}
              onChange={(e) => setChangeApplies(e.target.value)}
            >
              {changeAppliesOptions.map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
          </FormRow>

          <FormRow label={minLabel}>
            <input
              className="neon-input w-full"
              type="number"
              step={1}
              min={minBound}
              disabled={rangeNone}
              aria-label={minLabel}
              value={minValue}
              placeholder={intPlaceholder}
              onChange={(e) => setMinValue(toIntString(e.target.value))}
            />
          </FormRow>

          <FormRow label={maxLabel}>
            <input
              className="neon-input w-full"
              type="number"
              step={1}
              min={minBound}
              disabled={rangeNone}
              aria-label={maxLabel}
              value={maxValue}
              placeholder={intPlaceholder}
              onChange={(e) => setMaxValue(toIntString(e.target.value))}
            />
          </FormRow>
        </div>
      </div>
    </Modal>
  )
}
