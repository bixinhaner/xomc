import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  Pencil,
  Plus,
  RefreshCcw,
  Search,
  Trash2,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import {
  useNotificationTemplates,
  useDeleteNotificationTemplate,
} from '@core/hooks/api/useNotifications'
import type {
  NotificationChannel,
  NotificationLanguage,
  NotificationTemplate,
} from '@core/types/notification'

import { ChannelBadge, LANGUAGE_LABEL } from './shared'

// ============================================================
// 通知中心 → 通知模板列表（/notifications/templates）
// 对照 v1 webcode/src/pages/notifications/TemplateList。
//   - 渠道 / 语言 筛选 + 关键字（名称/主题）本地过滤
//   - 新建 → /notifications/templates/new
//   - 行内编辑 → /notifications/templates/:id（带参详情/编辑）
//   - 删除走 useDeleteNotificationTemplate（带二次确认）
// 数据全走真实 @core hooks，三态完整。
// ============================================================

const PAGE_SIZE = 10

type ChannelFilter = 'all' | NotificationChannel
type LanguageFilter = 'all' | NotificationLanguage

export default function NotificationTemplates() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [channel, setChannel] = useState<ChannelFilter>('all')
  const [language, setLanguage] = useState<LanguageFilter>('all')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(channel !== 'all' ? { channel } : {}),
      ...(language !== 'all' ? { language } : {}),
    }),
    [page, channel, language],
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useNotificationTemplates(params)
  const deleteMutation = useDeleteNotificationTemplate()

  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 关键字在当前页本地过滤（后端无 keyword 入参，名称/主题包含匹配）
  const rows = useMemo(() => {
    const all = data?.items ?? []
    const kw = keyword.trim().toLowerCase()
    if (!kw) return all
    return all.filter(
      (t) =>
        t.name.toLowerCase().includes(kw) ||
        t.subject.toLowerCase().includes(kw),
    )
  }, [data, keyword])

  const handleDelete = (t: NotificationTemplate) => {
    if (!window.confirm(`确认删除模板「${t.name}」？删除后不可恢复。`)) return
    deleteMutation.mutate(t.id)
  }

  const hasFilter = channel !== 'all' || language !== 'all' || keyword.trim() !== ''

  const cols = ['名称', '渠道', '语言', '主题', '状态', '更新时间', '操作']

  return (
    <PageShell
      title="通知模板"
      description={`共 ${total} 个通知模板`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/notifications')}
          >
            <ArrowLeft className="size-4" /> 返回
          </Button>

          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-60 pl-9"
              placeholder="搜索名称 / 主题"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>

          <Select
            value={channel}
            onValueChange={(v) => {
              setChannel(v as ChannelFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="渠道" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部渠道</SelectItem>
              <SelectItem value="email">邮件</SelectItem>
              <SelectItem value="sms">短信</SelectItem>
              <SelectItem value="webhook">Webhook</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={language}
            onValueChange={(v) => {
              setLanguage(v as LanguageFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="语言" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部语言</SelectItem>
              <SelectItem value="zh-CN">简体中文</SelectItem>
              <SelectItem value="en-US">English</SelectItem>
            </SelectContent>
          </Select>

          <div className="ml-auto flex items-center gap-2">
            <Button
              size="sm"
              onClick={() => navigate('/notifications/templates/new')}
            >
              <Plus className="size-4" /> 新建模板
            </Button>
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>
                {hasFilter ? '没有匹配的模板' : '暂无通知模板'}
              </EmptyRow>
            ) : (
              rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-medium text-primary hover:underline"
                      onClick={() =>
                        navigate(`/notifications/templates/${t.id}`)
                      }
                    >
                      {t.name}
                    </button>
                  </TableCell>
                  <TableCell>
                    <ChannelBadge channel={t.channel} />
                  </TableCell>
                  <TableCell className="text-xs">
                    {LANGUAGE_LABEL[t.language] ?? t.language}
                  </TableCell>
                  <TableCell className="max-w-xs truncate text-xs" title={t.subject}>
                    {t.subject || '—'}
                  </TableCell>
                  <TableCell>
                    {t.enabled ? (
                      <Badge variant="success">已启用</Badge>
                    ) : (
                      <Badge variant="muted">已禁用</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.updatedAt)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          navigate(`/notifications/templates/${t.id}`)
                        }
                      >
                        <Pencil className="size-4" /> 编辑
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={deleteMutation.isPending}
                        onClick={() => handleDelete(t)}
                      >
                        <Trash2 className="size-4 text-destructive" /> 删除
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      {deleteMutation.isError && (
        <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          删除失败：
          {deleteMutation.error instanceof Error
            ? deleteMutation.error.message
            : '未知错误'}
        </div>
      )}

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
