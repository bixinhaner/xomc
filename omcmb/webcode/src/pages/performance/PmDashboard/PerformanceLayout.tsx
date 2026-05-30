/**
 * T-0187 性能仪表盘入口页改写：顶部双页签 [任务仪表盘 | 设备列表]。
 *
 * 路由：`/performance?task=:id`（页签1 选中任务记 URL）。
 *
 * 页签1 · 任务仪表盘：
 *   - 左 240px 任务列表（内置区 / 自建区，usePmAdhocList isBuiltin 两次拉取，字段 is_builtin）。
 *   - 右侧 = <TaskDashboardPane taskId={selected} />，选中任务按 N 指标自动出图。
 * 页签2 · 设备列表：占位空态（建设中，T-0188 实现）。
 *
 * 旧拖拽编辑器 / 仪表盘列表 / KpiCardManager 在本入口不再 import（文件保留，清理见 T-0190）。
 */

import { useEffect, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Card, Empty, List, Space, Tabs, Tag, theme } from 'antd';
import { usePmAdhocList } from '@core/hooks/api/usePmAdhoc';
import type { AdhocTask } from '@core/types/pmAdhoc';
import TaskDashboardPane from './TaskDashboardPane';

interface TaskGroup {
  key: 'builtin' | 'custom';
  title: string;
  color: string;
  items: AdhocTask[];
}

function TaskDashboardTab() {
  const [searchParams, setSearchParams] = useSearchParams();
  const taskId = searchParams.get('task') ?? undefined;
  const { token } = theme.useToken();

  // 内置区 / 自建区两次独立拉取（字段 is_builtin），与 PmAdhoc 列表同源口径。
  const { data: builtinTasks = [], isLoading: builtinLoading } = usePmAdhocList({
    isBuiltin: true,
  });
  const { data: customTasks = [], isLoading: customLoading } = usePmAdhocList({
    isBuiltin: false,
  });
  const isLoading = builtinLoading || customLoading;

  const groups: TaskGroup[] = useMemo(
    () => [
      { key: 'builtin', title: '内置', color: 'blue', items: builtinTasks },
      { key: 'custom', title: '自建', color: 'green', items: customTasks },
    ],
    [builtinTasks, customTasks],
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

  return (
    <div style={{ display: 'flex', gap: 12, height: 'calc(100vh - 190px)' }}>
      <Card
        size="small"
        title="任务"
        style={{ width: 240, flexShrink: 0, overflow: 'auto' }}
        styles={{ body: { padding: 8 } }}
        loading={isLoading}
      >
        {groups.map((g) =>
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
        {allTasks.length === 0 && !isLoading && <Empty description="暂无聚合任务" />}
      </Card>

      <div style={{ flex: 1, overflow: 'auto' }}>
        {taskId ? (
          <TaskDashboardPane taskId={taskId} />
        ) : (
          <Card>
            <Empty description="左侧选择一个聚合任务查看自动出图" />
          </Card>
        )}
      </div>
    </div>
  );
}

function DeviceListTab() {
  return (
    <Card style={{ height: 'calc(100vh - 190px)' }}>
      <Empty description="设备列表（建设中）" style={{ marginTop: 80 }} />
    </Card>
  );
}

export default function PerformanceLayout() {
  return (
    <Tabs
      defaultActiveKey="task"
      items={[
        { key: 'task', label: '任务仪表盘', children: <TaskDashboardTab /> },
        { key: 'device', label: '设备列表', children: <DeviceListTab /> },
      ]}
    />
  );
}
