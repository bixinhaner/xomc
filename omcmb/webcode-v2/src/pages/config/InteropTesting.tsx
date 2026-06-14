import { useMemo, useState } from 'react'
import { CheckCircle2, Loader2, Play, RefreshCcw, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
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
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useTestCases, useRunTests } from '@core/hooks/api/useInterop'
import type { TestCase, RunTestsResponse } from '@core/services/api/interopApi'

// ============================================================
// 互操作测试 — 对照 v1 config/InteropTesting（F10）。
// 真实数据：useTestCases 用例库 + useRunTests 执行一致性测试。
// ============================================================

const CATEGORY_LABEL: Record<string, string> = {
  protocol: '协议',
  datamodel: '数据模型',
  rpc: 'RPC',
  inform: 'Inform',
}

const CATEGORY_VARIANT: Record<string, 'default' | 'success' | 'warning' | 'secondary'> = {
  protocol: 'default',
  datamodel: 'success',
  rpc: 'warning',
  inform: 'secondary',
}

const ALL_CATEGORIES = ['protocol', 'datamodel', 'rpc', 'inform']

export default function InteropTesting() {
  const { data: casesMap, isLoading, isError, error, isFetching, refetch } = useTestCases()
  const runTests = useRunTests()

  const [deviceSn, setDeviceSn] = useState('')
  const [category, setCategory] = useState('')
  const [results, setResults] = useState<RunTestsResponse | null>(null)
  const [runErr, setRunErr] = useState<string | null>(null)

  const allCases = useMemo<TestCase[]>(() => {
    if (!casesMap) return []
    return Object.values(casesMap).flat()
  }, [casesMap])

  const categoryCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const tc of allCases) counts[tc.category] = (counts[tc.category] ?? 0) + 1
    return counts
  }, [allCases])

  const handleRun = () => {
    if (!deviceSn.trim()) {
      setRunErr('请输入设备 SN')
      return
    }
    setRunErr(null)
    setResults(null)
    runTests.mutate(
      {
        deviceSn: deviceSn.trim(),
        categories: category ? [category] : undefined,
      },
      {
        onSuccess: (resp) => setResults(resp),
        onError: (e: Error) => setRunErr(e.message || '执行失败'),
      }
    )
  }

  const caseCols = ['用例 ID', '名称', '类别', '描述', '步骤数']
  const resultCols = ['用例', '类别', '结果', '耗时', '详情 / 错误']

  return (
    <PageShell
      title="互操作测试"
      description="设备一致性测试与数据模型校验（F10）"
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => void refetch()}>
          <RefreshCcw className="size-4" /> 刷新
        </Button>
      }
    >
      {/* 运行测试 */}
      <div className="mb-4 rounded-lg border bg-card p-4">
        <div className="mb-3 font-medium">运行一致性测试</div>
        {runErr && (
          <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {runErr}
          </div>
        )}
        <div className="flex flex-wrap items-end gap-3">
          <div>
            <Label className="mb-1 block text-xs">设备 SN</Label>
            <Input
              className="w-52"
              placeholder="ENB00001"
              value={deviceSn}
              onChange={(e) => setDeviceSn(e.target.value)}
            />
          </div>
          <div>
            <Label className="mb-1 block text-xs">类别（可选）</Label>
            <Select value={category || 'all'} onValueChange={(v) => setCategory(v === 'all' ? '' : v)}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="全部类别" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部类别</SelectItem>
                {ALL_CATEGORIES.map((c) => (
                  <SelectItem key={c} value={c}>
                    {CATEGORY_LABEL[c]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button disabled={runTests.isPending} onClick={handleRun}>
            {runTests.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Play className="size-4" />
            )}
            运行测试
          </Button>
        </div>
      </div>

      {/* 测试结果 */}
      {results && (
        <div className="mb-4 rounded-lg border bg-card p-4">
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <div className="font-medium">测试结果 · {results.deviceSn}</div>
            <Badge variant="secondary">总计 {results.total}</Badge>
            <Badge variant="success">通过 {results.passed}</Badge>
            <Badge variant={results.failed > 0 ? 'destructive' : 'muted'}>失败 {results.failed}</Badge>
            <Badge variant="muted">
              通过率 {results.total > 0 ? Math.round((results.passed / results.total) * 100) : 0}%
            </Badge>
          </div>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  {resultCols.map((c) => (
                    <TableHead key={c}>{c}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {results.results.length === 0 ? (
                  <EmptyRow colSpan={resultCols.length}>无结果</EmptyRow>
                ) : (
                  results.results.map((r) => (
                    <TableRow key={r.testCaseId} className={cn(!r.passed && 'bg-destructive/5')}>
                      <TableCell className="text-sm">{r.testName}</TableCell>
                      <TableCell>
                        <Badge variant={CATEGORY_VARIANT[r.category] ?? 'secondary'}>
                          {CATEGORY_LABEL[r.category] ?? r.category}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {r.passed ? (
                          <span className="inline-flex items-center gap-1 text-emerald-600 dark:text-emerald-400">
                            <CheckCircle2 className="size-4" /> 通过
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1 text-destructive">
                            <XCircle className="size-4" /> 失败
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-xs tabular-nums text-muted-foreground">
                        {r.durationMs}ms
                      </TableCell>
                      <TableCell className="max-w-md text-xs">
                        {r.error ? (
                          <span className="text-destructive">{r.error}</span>
                        ) : (
                          <span className="text-muted-foreground">{r.details || '—'}</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>
        </div>
      )}

      {/* 用例库 */}
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <div className="text-sm font-medium">测试用例库</div>
        {ALL_CATEGORIES.map((c) =>
          categoryCounts[c] ? (
            <Badge key={c} variant={CATEGORY_VARIANT[c] ?? 'secondary'}>
              {CATEGORY_LABEL[c]} {categoryCounts[c]}
            </Badge>
          ) : null
        )}
      </div>
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {caseCols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={caseCols.length} />
            ) : isError ? (
              <ErrorRow colSpan={caseCols.length} error={error} />
            ) : allCases.length === 0 ? (
              <EmptyRow colSpan={caseCols.length}>暂无测试用例</EmptyRow>
            ) : (
              allCases.map((tc) => (
                <TableRow key={tc.id}>
                  <TableCell className="font-mono text-xs">{tc.id}</TableCell>
                  <TableCell className="text-sm">{tc.name}</TableCell>
                  <TableCell>
                    <Badge variant={CATEGORY_VARIANT[tc.category] ?? 'secondary'}>
                      {CATEGORY_LABEL[tc.category] ?? tc.category}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-md truncate text-xs text-muted-foreground">
                    {tc.description || '—'}
                  </TableCell>
                  <TableCell className="tabular-nums">{tc.steps?.length ?? 0}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
