import { useState } from 'react';
import { Button, InputNumber, Progress, Table, message } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

interface LogConfigRow {
  id: string;
  logType: string;
  usedGB: number;
  totalGB: number;
  retentionDays: number;
  editing: boolean;
  editValue: number;
}

const initialConfigs: LogConfigRow[] = [
  { id: 'lc-001', logType: 'NE消息日志', usedGB: 12.5, totalGB: 50, retentionDays: 30, editing: false, editValue: 30 },
  { id: 'lc-002', logType: '心跳日志', usedGB: 3.2, totalGB: 20, retentionDays: 7, editing: false, editValue: 7 },
  { id: 'lc-003', logType: '操作日志', usedGB: 8.1, totalGB: 30, retentionDays: 90, editing: false, editValue: 90 },
  { id: 'lc-004', logType: '系统日志', usedGB: 6.8, totalGB: 30, retentionDays: 60, editing: false, editValue: 60 },
  { id: 'lc-005', logType: '告警操作日志', usedGB: 2.4, totalGB: 10, retentionDays: 180, editing: false, editValue: 180 },
  { id: 'lc-006', logType: '性能日志', usedGB: 28.7, totalGB: 100, retentionDays: 14, editing: false, editValue: 14 },
];

export default function LogConfig() {
  const t = useT();
  const [configs, setConfigs] = useState<LogConfigRow[]>(initialConfigs);

  const startEdit = (id: string) => {
    setConfigs((prev) => prev.map((c) => c.id === id ? { ...c, editing: true, editValue: c.retentionDays } : c));
  };

  const cancelEdit = (id: string) => {
    setConfigs((prev) => prev.map((c) => c.id === id ? { ...c, editing: false, editValue: c.retentionDays } : c));
  };

  const saveEdit = (id: string) => {
    setConfigs((prev) => prev.map((c) => {
      if (c.id !== id) return c;
      if (c.editValue < 1 || c.editValue > 180) {
        void message.error('保留天数范围为 1-180 天');
        return c;
      }
      void message.success(`${c.logType} 保留天数已更新为 ${c.editValue} 天`);
      return { ...c, retentionDays: c.editValue, editing: false };
    }));
  };

  const columns = [
    {
      title: t('table.type'),
      dataIndex: 'logType',
      key: 'logType',
      width: 160,
      render: (val: string) => <span style={{ fontWeight: 500 }}>{val}</span>,
    },
    {
      title: t('table.total'),
      dataIndex: 'usedGB',
      key: 'usedGB',
      width: 280,
      render: (_: unknown, record: LogConfigRow) => {
        const percent = Math.round((record.usedGB / record.totalGB) * 100);
        const status = percent > 85 ? 'exception' : percent > 70 ? 'normal' : 'success';
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <div style={{ flex: 1 }}>
              <Progress
                percent={percent}
                size="small"
                status={status}
                format={() => `${record.usedGB.toFixed(1)} / ${record.totalGB} GB`}
              />
            </div>
          </div>
        );
      },
    },
    {
      title: t('perf.timeRange'),
      dataIndex: 'retentionDays',
      key: 'retentionDays',
      width: 200,
      render: (_: unknown, record: LogConfigRow) => {
        if (record.editing) {
          return (
            <InputNumber
              value={record.editValue}
              min={1}
              max={180}
              addonAfter="天"
              style={{ width: 140 }}
              onChange={(val) => {
                if (val !== null) {
                  setConfigs((prev) => prev.map((c) => c.id === record.id ? { ...c, editValue: val } : c));
                }
              }}
            />
          );
        }
        return <span>{record.retentionDays} 天</span>;
      },
    },
    {
      title: t('table.operation'),
      key: 'actions',
      width: 80,
      fixed: 'right',
      render: (_: unknown, record: LogConfigRow) => {
        if (record.editing) {
          return (
            <div style={{ display: 'flex', gap: 8 }}>
              <Button
                type="primary"
                size="small"
                icon={<SaveOutlined />}
                onClick={() => saveEdit(record.id)}
              >
                {t('common.save')}
              </Button>
              <Button
                size="small"
                onClick={() => cancelEdit(record.id)}
              >
                {t('common.cancel')}
              </Button>
            </div>
          );
        }
        return (
          <Button
            type="link"
            size="small"
            onClick={() => startEdit(record.id)}
          >
            {t('common.edit')}
          </Button>
        );
      },
    },
  ];

  const storageTotal = configs.reduce((sum, c) => sum + c.totalGB, 0);
  const storageUsed = configs.reduce((sum, c) => sum + c.usedGB, 0);

  return (
    <ListPageLayout
      title={t('nav.log.config')}
      subtitle={t('nav.log.config')}
    >
      <div style={{ marginBottom: 12, padding: '12px 16px', background: '#f5f5f5', borderRadius: 6, display: 'flex', gap: 32 }}>
        <div>
          <span style={{ color: '#666', fontSize: 13 }}>{t('table.total')}:</span>
          <span style={{ fontWeight: 600 }}>{storageTotal} GB</span>
        </div>
        <div>
          <span style={{ color: '#666', fontSize: 13 }}>{t('table.total')}:</span>
          <span style={{ fontWeight: 600, color: storageUsed / storageTotal > 0.8 ? '#ff4d4f' : '#262626' }}>
            {storageUsed.toFixed(1)} GB ({Math.round((storageUsed / storageTotal) * 100)}%)
          </span>
        </div>
        <div>
          <span style={{ color: '#666', fontSize: 13 }}>{t('table.total')}:</span>
          <span style={{ fontWeight: 600 }}>{(storageTotal - storageUsed).toFixed(1)} GB</span>
        </div>
      </div>
      <Table
        dataSource={configs}
        columns={columns}
        rowKey="id"
        pagination={false}
        size="middle"
        bordered
      />
    </ListPageLayout>
  );
}
