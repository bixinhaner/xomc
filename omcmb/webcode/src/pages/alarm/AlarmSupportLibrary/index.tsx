import React, { useCallback, useMemo, useState } from 'react';
import { Button, Space, Tag, App } from 'antd';
import { ExportOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

// 告警库项类型定义
interface AlarmLibrary {
  deviceTypeName: string;    // 信告警源
  alarmIdentifier: string;   // 告警唯一标识
  alarmName: string;         // 可能原因
  serverityType: 'Critical' | 'Major' | 'Minor' | 'Warning';  // 严重程度
  eventType: EventType;      // 事件类型
  explanation: string;       // 告警解释
}

// 事件类型
type EventType = '30000' | '30001' | '30002' | '30003' | '30004' | '30006';

// Mock 数据
const MOCK_LIBRARY: AlarmLibrary[] = [
  {
    deviceTypeName: 'eNB',
    alarmIdentifier: 'ALM-0001',
    alarmName: 'CPU占用率超阈值告警',
    serverityType: 'Major',
    eventType: '30003',
    explanation: '当CPU占用率超过设定的阈值时产生此告警，可能导致系统性能下降。需要检查系统负载情况，必要时进行优化或扩容。',
  },
  {
    deviceTypeName: 'gNB',
    alarmIdentifier: 'ALM-0002',
    alarmName: '设备断连告警',
    serverityType: 'Critical',
    eventType: '30000',
    explanation: '设备与网管系统失去连接，可能是由于网络故障、设备断电或配置错误导致。需要立即检查设备状态和网络连接。',
  },
  {
    deviceTypeName: 'CPE',
    alarmIdentifier: 'ALM-0003',
    alarmName: '温度过高告警',
    serverityType: 'Major',
    eventType: '30004',
    explanation: '设备运行温度超过安全阈值，可能导致设备损坏或性能下降。需要检查机房散热条件和设备风扇状态。',
  },
  {
    deviceTypeName: 'eNB',
    alarmIdentifier: 'ALM-0004',
    alarmName: '光模块接收功率低',
    serverityType: 'Minor',
    eventType: '30000',
    explanation: '光模块接收光功率低于正常工作范围，可能是光纤损耗过大或光模块老化导致。建议检查光纤连接和光模块状态。',
  },
  {
    deviceTypeName: 'eGW',
    alarmIdentifier: 'ALM-0005',
    alarmName: '磁盘空间不足',
    serverityType: 'Warning',
    eventType: '30006',
    explanation: '磁盘空间使用率超过阈值，可能影响系统正常运行。建议清理日志文件或扩展存储容量。',
  },
  {
    deviceTypeName: 'gNB',
    alarmIdentifier: 'ALM-0006',
    alarmName: '链路丢包率异常',
    serverityType: 'Major',
    eventType: '30001',
    explanation: '链路丢包率超过正常范围，可能影响业务质量。需要检查网络链路质量和QoS配置。',
  },
  {
    deviceTypeName: 'eNB',
    alarmIdentifier: 'ALM-0007',
    alarmName: '软件版本不匹配',
    serverityType: 'Warning',
    eventType: '30002',
    explanation: '设备软件版本与预期版本不一致，可能导致功能异常。建议检查软件版本兼容性并进行升级或回退。',
  },
  {
    deviceTypeName: 'eGW',
    alarmIdentifier: 'ALM-0008',
    alarmName: '内存使用率超阈值',
    serverityType: 'Major',
    eventType: '30006',
    explanation: '内存使用率超过设定的阈值，可能导致系统性能下降或服务异常。需要检查内存占用情况并优化配置。',
  },
  {
    deviceTypeName: 'RRU',
    alarmIdentifier: 'ALM-0009',
    alarmName: '射频单元发射功率异常',
    serverityType: 'Critical',
    eventType: '30003',
    explanation: '射频单元发射功率超出正常范围，可能导致覆盖问题或干扰。需要立即检查射频单元状态和功率配置。',
  },
  {
    deviceTypeName: 'BBU',
    alarmIdentifier: 'ALM-0010',
    alarmName: '时钟同步异常',
    serverityType: 'Major',
    eventType: '30000',
    explanation: '基站时钟同步异常，可能影响切换性能和定位精度。需要检查时钟源配置和同步链路状态。',
  },
];

// 严重程度配置 - Material Design 颜色
const SEVERITY_CONFIG: Record<string, { color: string; bgColor: string }> = {
  Critical: { color: '#E53935', bgColor: '#FFEBEE' },
  Major: { color: '#FB8C00', bgColor: '#FFF3E0' },
  Minor: { color: '#FDD835', bgColor: '#FFFDE7' },
  Warning: { color: '#42A5F5', bgColor: '#E3F2FD' },
};

// 事件类型配置
const EVENT_TYPE_CONFIG: Record<EventType, string> = {
  '30000': '通信告警',
  '30001': '服务质量告警',
  '30002': '处理失败告警',
  '30003': '设备告警',
  '30004': '环境告警',
  '30006': '性能溢出告警',
};

export default function AlarmSupportLibrary() {
  const t = useT();
  const { message } = App.useApp();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [exportLoading, setExportLoading] = useState(false);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    Critical: t('alarm.severity.critical'),
    Major: t('alarm.severity.major'),
    Minor: t('alarm.severity.minor'),
    Warning: t('alarm.severity.warning'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    {
      name: 'keyword',
      label: t('alarm.search'),
      type: 'input',
      placeholder: t('alarm.librarySearchPlaceholder'),
    },
    {
      name: 'severity',
      label: t('alarm.severity'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.severity.critical'), value: 'Critical' },
        { label: t('alarm.severity.major'), value: 'Major' },
        { label: t('alarm.severity.minor'), value: 'Minor' },
        { label: t('alarm.severity.warning'), value: 'Warning' },
      ],
    },
    {
      name: 'eventType',
      label: t('alarm.eventType'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.eventType.communication'), value: '30000' },
        { label: t('alarm.eventType.qualityOfService'), value: '30001' },
        { label: t('alarm.eventType.processingError'), value: '30002' },
        { label: t('alarm.eventType.device'), value: '30003' },
        { label: t('alarm.eventType.environment'), value: '30004' },
        { label: t('alarm.eventType.performance'), value: '30006' },
      ],
    },
  ], [t]);

  // 过滤数据
  const filteredData = useMemo(() => {
    return MOCK_LIBRARY.filter((entry) => {
      if (filterParams.keyword) {
        const keyword = String(filterParams.keyword).toLowerCase();
        if (
          !entry.alarmIdentifier.toLowerCase().includes(keyword) &&
          !entry.alarmName.toLowerCase().includes(keyword) &&
          !entry.deviceTypeName.toLowerCase().includes(keyword)
        ) {
          return false;
        }
      }
      if (filterParams.severity && entry.serverityType !== filterParams.severity) {
        return false;
      }
      if (filterParams.eventType && entry.eventType !== filterParams.eventType) {
        return false;
      }
      return true;
    });
  }, [filterParams]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleExport = useCallback(async () => {
    setExportLoading(true);
    try {
      // TODO: 实现导出功能
      await new Promise((resolve) => setTimeout(resolve, 1000));
      message.success(t('common.exportSuccess'));
    } catch {
      message.error(t('common.exportFailed'));
    } finally {
      setExportLoading(false);
    }
  }, [t, message]);

  const columns = useMemo(
    (): DataTableColumn<AlarmLibrary>[] => [
      {
        key: 'deviceTypeName',
        title: t('alarm.deviceTypeName'),
        dataIndex: 'deviceTypeName',
        width: 110,
      },
      {
        key: 'alarmIdentifier',
        title: t('alarm.alarmIdentifier'),
        dataIndex: 'alarmIdentifier',
        width: 150,
        mono: true,
      },
      {
        key: 'alarmName',
        title: t('alarm.possibleCause'),
        dataIndex: 'alarmName',
        width: 300,
        ellipsis: true,
      },
      {
        key: 'serverityType',
        title: t('alarm.severity'),
        dataIndex: 'serverityType',
        width: 140,
        render: (value) => {
          const config = SEVERITY_CONFIG[value as string];
          return (
            <Tag style={{ color: config?.color, backgroundColor: config?.bgColor, border: 'none' }}>
              {SEVERITY_LABEL[value as string] || value}
            </Tag>
          );
        },
      },
      {
        key: 'eventType',
        title: t('alarm.eventType'),
        dataIndex: 'eventType',
        width: 160,
        render: (value) => t(`alarm.eventType.${EVENT_TYPE_TYPE_MAP[value as EventType]}`),
      },
      {
        key: 'explanation',
        title: t('alarm.explanation'),
        dataIndex: 'explanation',
        width: 700,
        ellipsis: true,
      },
    ],
    [t, SEVERITY_LABEL]
  );

  return (
    <ListPageLayout
      title={t('nav.alarm.library')}
      extra={
        <Button
          type="primary"
          icon={<ExportOutlined />}
          onClick={handleExport}
          loading={exportLoading}
        >
          {t('common.export')}
        </Button>
      }
    >
      <FilterBar
        filterId="alarm-library"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <DataTable<AlarmLibrary>
        tableId="alarm-library-table"
        columns={columns}
        dataSource={filteredData}
        loading={false}
        rowKey="alarmIdentifier"
        total={filteredData.length}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={(page, size) => {
          setCurrentPage(page);
          setPageSize(size);
        }}
        defaultDensity="compact"
      />
    </ListPageLayout>
  );
}

// 事件类型映射到 i18n key
const EVENT_TYPE_TYPE_MAP: Record<EventType, string> = {
  '30000': 'communication',
  '30001': 'qualityOfService',
  '30002': 'processingError',
  '30003': 'device',
  '30004': 'environment',
  '30006': 'performance',
};
