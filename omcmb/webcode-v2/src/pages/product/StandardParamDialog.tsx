import { useEffect, useMemo, useState } from 'react'

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

import {
  STANDARD_DATA_TYPES,
  STANDARD_CHANGE_APPLIES,
  dataTypeRangeKind,
  isUnsignedDataType,
} from '@core/types/paramModel'
import type { StandardParam, UpsertStandardInput } from '@core/types/paramModel'

// ============================================================
// ISSUE-488 (v2) — 标准参数树 新增 / 编辑弹窗。业务深度对齐 v1
// (webcode/src/pages/product/standard-params/index.tsx)：
//   字段 standardPath / entryType / access / dataType / changeApplies / min / max
//   - dataType / changeApplies 为下拉枚举（复用 frontend-core 共享常量），非文本框
//   - min/max label 随 dataType 动态（string→长度 / 数值→值），仅整数
//   - 编辑态历史值兼容：非枚举旧值并入下拉，不静默清空
// 设计语言：shadcn / Tailwind，弹窗用 fixed 覆盖层（对齐 v2 AlarmRuleDialog）。
// ============================================================

// InputNumber precision=0 等价：只接受整数，空串=不校验。
// 小数按截断（truncate）取整数段，而非删除小数点（避免 12.7→"127" 膨胀）。
function toIntString(raw: string): string {
  const trimmed = raw.trim()
  if (trimmed === '') return ''
  // 保留前导负号（signed int 的负 min/max 是合法语义）。
  const negative = trimmed.startsWith('-')
  // 按小数点切分，只取整数段；再剔除整数段里的非数字字符。
  const intPart = trimmed.replace(/^-/, '').split('.')[0].replace(/[^\d]/g, '')
  if (intPart === '') return ''
  // Number(...) 归一化前导零等；符号只在此处拼一次，避免 "--5"。
  return String(Number((negative ? '-' : '') + intPart))
}

export function StandardParamDialog({
  open,
  editing,
  loading,
  onSubmit,
  onCancel,
}: {
  open: boolean
  editing: StandardParam | null
  loading?: boolean
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

  if (!open) return null

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
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onCancel} aria-hidden />
      <div className="relative w-full max-w-lg rounded-lg border bg-background p-5 shadow-xl">
        <h2 className="text-base font-semibold">
          {editing ? '编辑标准参数' : '新增标准参数'}
        </h2>

        <div className="mt-4 space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="std-path">标准路径</Label>
            <Input
              id="std-path"
              value={standardPath}
              placeholder="例如 InternetGatewayDevice.DeviceInfo.SoftwareVersion"
              disabled={Boolean(editing)}
              onChange={(e) => {
                setStandardPath(e.target.value)
                if (e.target.value.trim() !== '') setPathErr(false)
              }}
              autoFocus={!editing}
            />
            {pathErr ? (
              <p className="text-xs text-destructive">标准路径必填</p>
            ) : null}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="std-entry">条目类型</Label>
              <Select value={entryType} onValueChange={setEntryType}>
                <SelectTrigger id="std-entry">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="parameter">parameter</SelectItem>
                  <SelectItem value="object">object</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="std-access">访问</Label>
              <Select value={access} onValueChange={setAccess}>
                <SelectTrigger id="std-access">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="readWrite">readWrite</SelectItem>
                  <SelectItem value="readOnly">readOnly</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="std-datatype">数据类型</Label>
              <Select
                value={dataType}
                onValueChange={(v) => {
                  setDataType(v)
                  // 切到无取值范围类型（boolean/dateTime）时清空 min/max。
                  if (dataTypeRangeKind(v) === 'none') {
                    setMinValue('')
                    setMaxValue('')
                  }
                }}
              >
                <SelectTrigger id="std-datatype">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {dataTypeOptions.map((v) => (
                    <SelectItem key={v} value={v}>
                      {v}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="std-change">生效方式</Label>
              <Select value={changeApplies} onValueChange={setChangeApplies}>
                <SelectTrigger id="std-change">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {changeAppliesOptions.map((v) => (
                    <SelectItem key={v} value={v}>
                      {v}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="std-min">{minLabel}</Label>
              <Input
                id="std-min"
                type="number"
                step={1}
                min={minBound}
                disabled={rangeNone}
                value={minValue}
                placeholder={intPlaceholder}
                onChange={(e) => setMinValue(toIntString(e.target.value))}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="std-max">{maxLabel}</Label>
              <Input
                id="std-max"
                type="number"
                step={1}
                min={minBound}
                disabled={rangeNone}
                value={maxValue}
                placeholder={intPlaceholder}
                onChange={(e) => setMaxValue(toIntString(e.target.value))}
              />
            </div>
          </div>
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={loading}>
            取消
          </Button>
          <Button size="sm" onClick={submit} disabled={loading}>
            {editing ? '保存' : '创建'}
          </Button>
        </div>
      </div>
    </div>
  )
}
