import { Button, Card, Empty, Popconfirm, Space, Tag, Tooltip, Typography } from 'antd';
import { DeleteOutlined, HistoryOutlined, LeftOutlined, RightOutlined } from '@ant-design/icons';
import type { ExecRecord } from '../types';
import { opColor } from '../constants';

const { Text } = Typography;

interface CommandHistoryPanelProps {
  records: ExecRecord[];
  activeId: string | null;
  collapsed: boolean;
  onSelect: (id: string) => void;
  /** 清空命令记录（清内存列表 + localStorage 的命令 ID 数组，§3.11.4） */
  onClear: () => void;
  onToggleCollapsed: () => void;
}

/**
 * 执行命令记录列表（设计 §3.10.4~6 + §3.11.4）—— 每条含时间 / 命令名(op 彩 Tag) / 设备数 /
 * 命令 ID 深链任务详情。可靠左收缩(默认收缩为窄条)；点击某条 → 右侧执行结果联动切换(§3.10.5)；
 * 标题栏「清空」清记录(localStorage 只存命令 ID)。
 */
export default function CommandHistoryPanel({
  records,
  activeId,
  collapsed,
  onSelect,
  onClear,
  onToggleCollapsed,
}: CommandHistoryPanelProps) {
  // 收缩态:窄条,仅图标 + 展开按钮 + 记录数徽标。
  if (collapsed) {
    return (
      <div
        style={{
          height: '100%',
          width: 44,
          flex: '0 0 44px',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 12,
          paddingTop: 12,
          background: 'var(--color-bg-container)',
          border: '1px solid #f0f0f0',
          borderRadius: 8,
        }}
      >
        <Tooltip title="展开命令记录" placement="right">
          <Button type="text" icon={<RightOutlined />} onClick={onToggleCollapsed} />
        </Tooltip>
        <HistoryOutlined style={{ fontSize: 18, color: '#8c8c8c' }} />
        <Text type="secondary" style={{ writingMode: 'vertical-rl', letterSpacing: 4, marginTop: 4 }}>
          命令记录
        </Text>
      </div>
    );
  }

  return (
    <Card
      size="small"
      title={
        <Space size={6}>
          <HistoryOutlined />
          <span>命令记录</span>
        </Space>
      }
      extra={
        <Space size={2}>
          <Popconfirm
            title="清空命令记录"
            description="将清除本地缓存的命令记录列表（不影响服务端任务）。"
            okText="清空"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={onClear}
            disabled={records.length === 0}
          >
            <Tooltip title="清空命令记录">
              <Button
                type="text"
                size="small"
                icon={<DeleteOutlined />}
                disabled={records.length === 0}
              />
            </Tooltip>
          </Popconfirm>
          <Tooltip title="收起">
            <Button type="text" size="small" icon={<LeftOutlined />} onClick={onToggleCollapsed} />
          </Tooltip>
        </Space>
      }
      style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
      styles={{ body: { flex: 1, minHeight: 0, overflow: 'auto', padding: 8 } }}
    >
      {records.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无执行记录" style={{ marginTop: 48 }} />
      ) : (
        <Space direction="vertical" size={6} style={{ width: '100%' }}>
          {records.map((r) => {
            const active = r.id === activeId;
            return (
              <div
                key={r.id}
                onClick={() => onSelect(r.id)}
                style={{
                  cursor: 'pointer',
                  padding: '8px 10px',
                  borderRadius: 6,
                  border: active ? '1px solid #1677ff' : '1px solid #f0f0f0',
                  background: active ? 'rgba(22,119,255,0.06)' : 'transparent',
                }}
              >
                <Space size={6} style={{ width: '100%', justifyContent: 'space-between' }}>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {r.time}
                  </Text>
                  {/* 不再深链「任务详情」：一条命令记录对应多条「分设备」任务记录，
                      单 task 详情页无法对应；逐设备详情用结果表格行「查看」。 */}
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {r.deviceCount} 台
                  </Text>
                </Space>
                <div style={{ marginTop: 4 }}>
                  <Tag color={opColor(r.operationType)} style={{ marginInlineEnd: 6 }}>
                    {r.operationType}
                  </Tag>
                  <Text ellipsis={{ tooltip: r.commandName }} style={{ fontSize: 13 }}>
                    {r.commandName}
                  </Text>
                </div>
              </div>
            );
          })}
        </Space>
      )}
    </Card>
  );
}
