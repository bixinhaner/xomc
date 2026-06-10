/**
 * 性能仪表盘入口页（任务仪表盘）。
 *
 * 路由：`/performance?task=:id`（选中任务记 URL）。
 *   - 左 240px 任务列表（内置区 / 自建区，usePmAdhocList isBuiltin 两次拉取，字段 is_builtin）。
 *   - 右侧 = <TaskDashboardPane taskId={selected} />，选中任务按 N 指标自动出图。
 *
 * 原「设备列表」页签已拆为性能管理独立子菜单「设备性能查看」（路由 /performance/device-view，
 * 组件仍是 DeviceListPane），本页不再承载页签，直接渲染任务仪表盘。
 */

import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useIntl } from 'react-intl';
import { Button, Card, Empty, List, Space, Tag, theme, Tooltip } from 'antd';
import { MenuFoldOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import { usePmAdhocList } from '@core/hooks/api/usePmAdhoc';
import type { AdhocTask } from '@core/types/pmAdhoc';
import EmptyState from '@/components/common/EmptyState';
import ErrorBoundary from '@/components/common/ErrorBoundary';
import TaskDashboardPane from './TaskDashboardPane';

interface TaskGroup {
  key: 'builtin' | 'custom';
  title: string;
  color: string;
  items: AdhocTask[];
}

function TaskDashboardTab() {
  const intl = useIntl();
  const [searchParams, setSearchParams] = useSearchParams();
  const taskId = searchParams.get('task') ?? undefined;
  const { token } = theme.useToken();

  // 内置区 / 自建区两次独立拉取（字段 is_builtin），与 PmAdhoc 列表同源口径。
  const {
    data: builtinTasks = [],
    isLoading: builtinLoading,
    isError: builtinError,
    refetch: refetchBuiltin,
  } = usePmAdhocList({ isBuiltin: true });
  const {
    data: customTasks = [],
    isLoading: customLoading,
    isError: customError,
    refetch: refetchCustom,
  } = usePmAdhocList({ isBuiltin: false });
  const isLoading = builtinLoading || customLoading;
  const isError = builtinError || customError;

  const groups: TaskGroup[] = useMemo(
    () => [
      {
        key: 'builtin',
        title: intl.formatMessage({ id: 'perf.dashboard.groupBuiltin' }),
        color: 'blue',
        items: builtinTasks,
      },
      {
        key: 'custom',
        title: intl.formatMessage({ id: 'perf.dashboard.groupCustom' }),
        color: 'green',
        items: customTasks,
      },
    ],
    [builtinTasks, customTasks, intl],
  );

  const allTasks = useMemo(
    () => [...builtinTasks, ...customTasks],
    [builtinTasks, customTasks],
  );

  // 默认选中：URL 未指定 / 指向的任务已不存在 → 回退首个内置（再退化自建第一个）。
  // 放 useEffect 而非 render body，避免 React 19 严格模式「渲染中更新组件」告警。
  const selectedMissing = Boolean(taskId) && !allTasks.some((t) => t.id === taskId);
  useEffect(() => {
    if (!isLoading && allTasks.length > 0 && (!taskId || selectedMissing)) {
      const first = builtinTasks[0] ?? customTasks[0];
      if (first && first.id !== taskId) {
        setSearchParams({ task: first.id }, { replace: true });
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, allTasks, taskId, selectedMissing]);

  const handleSelect = (t: AdhocTask) => {
    setSearchParams({ task: t.id });
  };

  // 任务栏收起状态（仅本页本地状态，刷新后回到展开）。
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div style={{ display: 'flex', gap: 12, height: 'calc(100vh - 190px)' }}>
      {collapsed ? (
        <Tooltip title={intl.formatMessage({ id: 'perf.dashboard.expandTaskPane' })} placement="right">
          <div
            onClick={() => setCollapsed(false)}
            style={{
              width: 36,
              flexShrink: 0,
              cursor: 'pointer',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 8,
              paddingTop: 8,
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusLG,
              background: token.colorBgContainer,
            }}
          >
            <MenuUnfoldOutlined style={{ color: token.colorPrimary }} />
            <span
              style={{
                fontSize: 12,
                color: token.colorTextSecondary,
                writingMode: 'vertical-rl',
                letterSpacing: 2,
              }}
            >
              {intl.formatMessage({ id: 'perf.dashboard.taskListTitle' })}
            </span>
          </div>
        </Tooltip>
      ) : (
      <Card
        size="small"
        title={intl.formatMessage({ id: 'perf.dashboard.taskListTitle' })}
        extra={
          <Tooltip title={intl.formatMessage({ id: 'perf.dashboard.collapseTaskPane' })}>
            <Button
              type="text"
              size="small"
              icon={<MenuFoldOutlined />}
              onClick={() => setCollapsed(true)}
              aria-label={intl.formatMessage({ id: 'perf.dashboard.collapseTaskPane' })}
            />
          </Tooltip>
        }
        style={{ width: 240, flexShrink: 0, overflow: 'auto' }}
        styles={{ body: { padding: 8 } }}
        loading={isLoading}
      >
        {isError && (
          <EmptyState
            variant="error"
            style={{ padding: '24px 0' }}
            action={{
              label: intl.formatMessage({ id: 'error.retry' }),
              onClick: () => {
                void refetchBuiltin();
                void refetchCustom();
              },
            }}
          />
        )}
        {!isError && groups.map((g) =>
          g.items.length === 0 ? null : (
            <div key={g.key} style={{ marginBottom: 12 }}>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 4, paddingLeft: 4 }}>
                <Tag color={g.color}>{g.title}</Tag>
                <span style={{ marginLeft: 4 }}>{g.items.length}</span>
              </div>
              <List
                size="small"
                dataSource={g.items}
                renderItem={(t) => {
                  const active = t.id === taskId;
                  return (
                    <List.Item
                      onClick={() => handleSelect(t)}
                      style={{
                        cursor: 'pointer',
                        padding: '6px 8px',
                        background: active ? token.controlItemBgActive : undefined,
                        color: active ? token.colorPrimary : undefined,
                        borderRadius: 4,
                      }}
                    >
                      <Space size={4} style={{ width: '100%' }}>
                        {t.technology && (
                          <Tag style={{ marginRight: 0 }}>{t.technology.toUpperCase()}</Tag>
                        )}
                        <span
                          style={{
                            flex: 1,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {t.name}
                        </span>
                      </Space>
                    </List.Item>
                  );
                }}
              />
            </div>
          ),
        )}
        {allTasks.length === 0 && !isLoading && !isError && (
          <Empty description={intl.formatMessage({ id: 'perf.dashboard.emptyNoTask' })} />
        )}
      </Card>
      )}

      <div style={{ flex: 1, overflow: 'auto' }}>
        {taskId ? (
          // 细粒度 ErrorBoundary：单个任务图表渲染异常时只影响右侧面板，左侧任务列表仍可切换。
          <ErrorBoundary key={taskId}>
            <TaskDashboardPane taskId={taskId} />
          </ErrorBoundary>
        ) : (
          <Card>
            <Empty description={intl.formatMessage({ id: 'perf.dashboard.emptySelectTask' })} />
          </Card>
        )}
      </div>
    </div>
  );
}

export default function PerformanceLayout() {
  return <TaskDashboardTab />;
}
