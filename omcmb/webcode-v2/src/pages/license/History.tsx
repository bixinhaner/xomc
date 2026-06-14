/**
 * SystemLicenseHistory (webcode-v2) — 许可证历史子页。
 *
 * 对齐 v1 webcode/src/pages/SystemLicense/History.tsx，路由 /license/history。
 * 后端 useSystemLicenseHistory 默认按 replaced_at DESC 返回所有被替换过的归档 license。
 *
 * 列：License ID / 类型 / 签名状态 / 上传时间 / 替换时间。
 * 支持按 License ID 过滤（API 的可选 licenseId 参数）。
 * 三态完整：loading / error / empty；分页走 PageShell.Pagination。
 *
 * UI 走 shadcn/Tailwind（浅色），数据全走 @core React Query hook，不新增契约。
 */
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, RefreshCcw, Search } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useSystemLicenseHistory } from '@core/hooks/api/useSystemLicense'
import type {
  SystemLicenseHistoryQuery,
} from '@core/services/api/systemLicenseApi'
import type { SystemLicenseSignatureStatus } from '@core/services/api/systemLicenseApi'

const SIGNATURE_VARIANT: Record<
  SystemLicenseSignatureStatus,
  'success' | 'destructive' | 'warning'
> = {
  verified: 'success',
  invalid: 'destructive',
  unverified: 'warning',
}

const SIGNATURE_LABEL: Record<SystemLicenseSignatureStatus, string> = {
  verified: '已验证',
  invalid: '签名无效',
  unverified: '未验证',
}

const COL_COUNT = 6

export default function SystemLicenseHistory() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [licenseIdInput, setLicenseIdInput] = useState('')
  const [licenseIdFilter, setLicenseIdFilter] = useState('')

  const params = useMemo<SystemLicenseHistoryQuery>(
    () => ({
      page,
      pageSize,
      ...(licenseIdFilter.trim() ? { licenseId: licenseIdFilter.trim() } : {}),
    }),
    [page, pageSize, licenseIdFilter],
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useSystemLicenseHistory(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const applyFilter = () => {
    setLicenseIdFilter(licenseIdInput)
    setPage(1)
  }

  return (
    <PageShell
      title="许可证历史"
      description="系统许可证每次更新后保留的归档记录（按替换时间倒序）"
      isFetching={isFetching}
      toolbar={
        <>
          <Button variant="outline" size="sm" onClick={() => navigate('/license')}>
            <ArrowLeft /> 返回许可证
          </Button>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="按 License ID 过滤"
              value={licenseIdInput}
              onChange={(e) => setLicenseIdInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applyFilter()
              }}
            />
          </div>
          <Button variant="outline" size="sm" onClick={applyFilter}>
            <Search /> 查询
          </Button>
          <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
            <span>共 {total} 条</span>
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw /> 刷新
            </Button>
          </div>
        </>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>License ID</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>签名状态</TableHead>
              <TableHead>上传时间</TableHead>
              <TableHead>替换时间</TableHead>
              <TableHead>记录 PK</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={COL_COUNT} />
            ) : isError ? (
              <ErrorRow colSpan={COL_COUNT} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={COL_COUNT}>暂无历史记录</EmptyRow>
            ) : (
              rows.map((h) => (
                <TableRow key={h.id}>
                  <TableCell className="font-mono text-xs">{h.licenseId}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{h.licenseType}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={SIGNATURE_VARIANT[h.signatureStatus]}>
                      {SIGNATURE_LABEL[h.signatureStatus]}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(h.uploadedAt)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(h.replacedAt)}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {h.id}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={pageSize}
        onChange={setPage}
      />
    </PageShell>
  )
}
