import { useState } from 'react'
import { RotateCcw, Save } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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

// ============================================================
// 拓扑设置（v2 皮肤）— 对照 v1 webcode/pages/topology/TopologySettings
//  v1 本就是纯前端可视化偏好（布局算法 / 节点样式 / 边样式 / 显示 / 交互），
//  保存仅本地（无后端接口）。v2 复刻同样的纯本地表单：受控 state + 保存到内存
//  + 恢复默认。无异步数据，故不涉及 loading/error 三态。
// ============================================================

interface SettingsValues {
  layoutAlgorithm: 'force' | 'tree' | 'circular' | 'hierarchy'
  nodeSize: number
  nodeShape: 'circle' | 'rect' | 'diamond'
  showNodeLabel: boolean
  edgeWidth: number
  edgeStyle: 'solid' | 'dashed' | 'dotted'
  showEdgeLabel: boolean
  showStatusBadge: boolean
  enableAnimation: boolean
  autoRefreshInterval: number
}

const DEFAULTS: SettingsValues = {
  layoutAlgorithm: 'force',
  nodeSize: 24,
  nodeShape: 'circle',
  showNodeLabel: true,
  edgeWidth: 2,
  edgeStyle: 'solid',
  showEdgeLabel: false,
  showStatusBadge: true,
  enableAnimation: true,
  autoRefreshInterval: 30,
}

const REFRESH_OPTIONS = [
  { label: '不自动刷新', value: 0 },
  { label: '10 秒', value: 10 },
  { label: '30 秒', value: 30 },
  { label: '60 秒', value: 60 },
  { label: '5 分钟', value: 300 },
]

function Toggle({
  checked,
  onChange,
}: {
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
        checked ? 'bg-primary' : 'bg-muted'
      }`}
    >
      <span
        className={`inline-block size-4 transform rounded-full bg-white shadow transition-transform ${
          checked ? 'translate-x-4' : 'translate-x-0.5'
        }`}
      />
    </button>
  )
}

export default function TopologySettings() {
  const [values, setValues] = useState<SettingsValues>(DEFAULTS)
  const [saved, setSaved] = useState(false)

  const set = <K extends keyof SettingsValues>(key: K, val: SettingsValues[K]) => {
    setValues((prev) => ({ ...prev, [key]: val }))
    setSaved(false)
  }

  const handleSave = () => {
    // 纯前端偏好：保存进内存即可（v1 同样无后端接口）。
    setSaved(true)
  }

  const handleReset = () => {
    setValues(DEFAULTS)
    setSaved(false)
  }

  return (
    <PageShell
      title="拓扑设置"
      description="拓扑图可视化偏好（布局 / 节点 / 边 / 显示 / 交互）"
      toolbar={
        <div className="flex w-full items-center gap-2">
          {saved && <span className="text-xs text-emerald-600 dark:text-emerald-400">已保存</span>}
          <div className="ml-auto flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleReset}>
              <RotateCcw className="mr-1 h-4 w-4" />
              恢复默认
            </Button>
            <Button size="sm" onClick={handleSave}>
              <Save className="mr-1 h-4 w-4" />
              保存
            </Button>
          </div>
        </div>
      }
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* 布局 */}
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">布局算法</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 p-4 pt-2">
            <div className="space-y-1.5">
              <Label>布局算法</Label>
              <Select
                value={values.layoutAlgorithm}
                onValueChange={(v) => set('layoutAlgorithm', v as SettingsValues['layoutAlgorithm'])}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="force">力导向</SelectItem>
                  <SelectItem value="tree">树形</SelectItem>
                  <SelectItem value="circular">环形</SelectItem>
                  <SelectItem value="hierarchy">分层</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>自动刷新间隔</Label>
              <Select
                value={String(values.autoRefreshInterval)}
                onValueChange={(v) => set('autoRefreshInterval', Number(v))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {REFRESH_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={String(o.value)}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* 节点样式 */}
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">节点样式</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 p-4 pt-2">
            <div className="space-y-1.5">
              <Label>节点尺寸</Label>
              <Input
                type="number"
                min={12}
                max={48}
                value={values.nodeSize}
                onChange={(e) => set('nodeSize', Number(e.target.value))}
              />
            </div>
            <div className="space-y-1.5">
              <Label>节点形状</Label>
              <Select
                value={values.nodeShape}
                onValueChange={(v) => set('nodeShape', v as SettingsValues['nodeShape'])}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="circle">圆形</SelectItem>
                  <SelectItem value="rect">矩形</SelectItem>
                  <SelectItem value="diamond">菱形</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center justify-between">
              <Label>显示节点标签</Label>
              <Toggle checked={values.showNodeLabel} onChange={(v) => set('showNodeLabel', v)} />
            </div>
          </CardContent>
        </Card>

        {/* 边样式 */}
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">边样式</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 p-4 pt-2">
            <div className="space-y-1.5">
              <Label>边宽度</Label>
              <Input
                type="number"
                min={1}
                max={6}
                value={values.edgeWidth}
                onChange={(e) => set('edgeWidth', Number(e.target.value))}
              />
            </div>
            <div className="space-y-1.5">
              <Label>线型</Label>
              <Select
                value={values.edgeStyle}
                onValueChange={(v) => set('edgeStyle', v as SettingsValues['edgeStyle'])}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="solid">实线</SelectItem>
                  <SelectItem value="dashed">虚线</SelectItem>
                  <SelectItem value="dotted">点线</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center justify-between">
              <Label>显示边标签</Label>
              <Toggle checked={values.showEdgeLabel} onChange={(v) => set('showEdgeLabel', v)} />
            </div>
          </CardContent>
        </Card>

        {/* 显示与交互 */}
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">显示与交互</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 p-4 pt-2">
            <div className="flex items-center justify-between">
              <Label>显示状态徽标</Label>
              <Toggle checked={values.showStatusBadge} onChange={(v) => set('showStatusBadge', v)} />
            </div>
            <div className="flex items-center justify-between">
              <Label>启用动画</Label>
              <Toggle checked={values.enableAnimation} onChange={(v) => set('enableAnimation', v)} />
            </div>
          </CardContent>
        </Card>
      </div>
    </PageShell>
  )
}
