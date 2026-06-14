import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { EmptyRow, PageShell, TableCard } from '@/components/layout/PageShell'

import { useProductDetail } from '@core/hooks/api/useProducts'
import type { ProductPattern } from '@core/types/product'

// ============================================================
// 产品装配件详情 — 基本信息 + 匹配正则清单(按 sortOrder)。
// 路由 /product/products/:id,经 useProductDetail 真实加载。
// ============================================================

export function ProductDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError, error, isFetching } = useProductDetail(id)

  const product = data?.product
  const patterns = [...(data?.patterns ?? [])].sort((a, b) => a.sortOrder - b.sortOrder)

  return (
    <PageShell
      title={product ? product.name : '产品详情'}
      description={id ? `产品 ID ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/product/products')}>
          <ArrowLeft className="size-4" /> 返回清单
        </Button>
      }
    >
      {isLoading ? (
        <Card>
          <CardContent className="py-16 text-center text-muted-foreground">加载中…</CardContent>
        </Card>
      ) : isError ? (
        <Card>
          <CardContent className="py-16 text-center text-destructive">
            加载失败：{error instanceof Error ? error.message : '未知错误'}
          </CardContent>
        </Card>
      ) : !product ? (
        <Card>
          <CardContent className="py-16 text-center text-muted-foreground">未找到产品</CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                基本信息
                {product.isBuiltin && <Badge variant="muted">内置</Badge>}
                <Badge variant="outline">{(product.tech || '—').toUpperCase()}</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm md:grid-cols-3">
                <Field label="厂商" value={product.vendor} />
                <Field label="制式" value={product.tech} />
                <Field label="射频模式" value={product.radioModes} />
                <Field label="参数模型库" value={product.paramModelName} />
                <Field label="指标设备类型" value={product.indicatorDeviceType} />
                <Field label="指标平台" value={product.indicatorPlatform} />
                <Field label="告警网元类型" value={product.alarmNeType} />
                <Field label="关联设备数" value={String(product.deviceCount ?? 0)} />
                <Field
                  label="未知告警"
                  value={product.enableUnknownAlarm ? '接收' : '丢弃'}
                />
                <Field
                  label="文件类型 11"
                  value={product.enableFiletype11 ? '启用' : '停用'}
                />
                <div className="col-span-2 md:col-span-3">
                  <dt className="text-xs uppercase tracking-wider text-muted-foreground">描述</dt>
                  <dd className="mt-1">{product.description || '—'}</dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          <div>
            <h2 className="mb-2 text-sm font-semibold">
              匹配正则 <span className="text-muted-foreground">({patterns.length})</span>
            </h2>
            <TableCard>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">顺序</TableHead>
                    <TableHead>正则 (productClass)</TableHead>
                    <TableHead className="w-24">来源</TableHead>
                    <TableHead className="w-24">状态</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {patterns.length === 0 ? (
                    <EmptyRow colSpan={4}>暂无匹配正则</EmptyRow>
                  ) : (
                    patterns.map((p) => <PatternRow key={p.id} p={p} />)
                  )}
                </TableBody>
              </Table>
            </TableCard>
          </div>
        </div>
      )}
    </PageShell>
  )
}

function PatternRow({ p }: { p: ProductPattern }) {
  return (
    <TableRow>
      <TableCell className="tabular-nums text-muted-foreground">{p.sortOrder}</TableCell>
      <TableCell className="font-mono text-xs">{p.productClass}</TableCell>
      <TableCell>
        <Badge variant={p.source === 'custom' ? 'default' : 'muted'}>
          {p.source === 'custom' ? '自定义' : '内置'}
        </Badge>
      </TableCell>
      <TableCell>
        {p.isActive ? (
          <Badge variant="success">启用</Badge>
        ) : (
          <Badge variant="muted">停用</Badge>
        )}
      </TableCell>
    </TableRow>
  )
}

function Field({ label, value }: { label: string; value?: string | null }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="mt-1">{value || '—'}</dd>
    </div>
  )
}

export default ProductDetailPage
