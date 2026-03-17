import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Modal, Space, Tag, Typography, message } from 'antd';
import { DeleteOutlined, DownloadOutlined, EditOutlined, EyeOutlined, PlusOutlined, SettingOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import StatisticsPanel from '@/components/StatisticsPanel';
import StatusIndicator from '@/components/StatusIndicator';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useDeviceList, useDeleteDevices } from '@/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { Device } from '@/types/device';

const { Link } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};


export default function DeviceList() {
  const t = useT();
  const navigate = useNavigate();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const queryParams = useMemo(
    () => ({ ...filterParams, page: currentPage, pageSize } as Parameters<typeof useDeviceList>[0]),
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useDeviceList(queryParams);
  const deleteDevices = useDeleteDevices();

  const devices: Device[] = data?.items ?? [];
  const total = data?.total ?? 0;

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
    none: t('alarm.severity.none'),
  }), [t]);


  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'sn', label: t('device.sn'), type: 'input' },
    { name: 'hostName', label: t('device.hostName'), type: 'input' },
    {
      name: 'networkType',
      label: t('device.radioMode'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'GSM', value: 'GSM' },
      ],
    },
    {
      name: 'productType',
      label: t('device.productType'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'GSM', value: 'GSM' },
      ],
    },
    {
      name: 'connStatus',
      label: t('device.connStatus'),
      type: 'select',
      options: [
        { label: t('status.online'), value: 'online' },
        { label: t('status.offline'), value: 'offline' },
      ],
    },
    {
      name: 'opState',
      label: t('device.opState'),
      type: 'select',
      options: [
        { label: t('status.active'), value: 'active' },
        { label: t('status.inactive'), value: 'inactive' },
      ],
    },
    {
      name: 'groupName',
      label: t('device.groupName'),
      type: 'input',
    },
  ], [t]);

  // Statistics
  const statsItems = useMemo(() => {
    const online = devices.filter((d) => d.connStatus === 'online').length;
    const offline = devices.filter((d) => d.connStatus === 'offline').length;
    const alarmed = devices.filter((d) => d.alarmLevel !== 'none').length;
    return [
      { label: t('device.count.total'), value: total },
      { label: t('status.online'), value: online, color: '#52C41A' },
      { label: t('status.offline'), value: offline, color: '#8C8C8C' },
      { label: t('common.hasAlarm'), value: alarmed, color: '#FA8C16' },
    ];
  }, [devices, total, t]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleDelete = useCallback(
    (ids: string[]) => {
      Modal.confirm({
        title: t('common.confirmDelete'),
        content: t('common.deleteConfirmMsg', { count: ids.length }),
        okText: t('common.confirmDelete'),
        okType: 'danger',
        cancelText: t('common.cancel'),
        onOk: async () => {
          await deleteDevices.mutateAsync(ids);
          void message.success(t('common.deleteSuccess'));
          setSelectedRowKeys([]);
        },
      });
    },
    [deleteDevices, t]
  );

  // 格式化时间戳
  const fmtTime = useCallback((v: string) => (v ? new Date(v).toLocaleString('zh-CN') : '-'), []);

  // 格式化在线时长(秒)
  const fmtDuration = useCallback((seconds: number) => {
    if (!seconds) return '-';
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    return d > 0 ? `${d}d ${h}h ${m}m` : h > 0 ? `${h}h ${m}m` : `${m}m`;
  }, []);

  const columns = useMemo(
    (): DataTableColumn<Device>[] => [
      // =====================================================================
      // 公共字段 (common) — 三制式共有
      // =====================================================================
      {
        key: 'sn',
        title: 'SN',
        dataIndex: 'sn',
        width: 180,
        mono: true,
        copyable: true,
        group: 'common',
        render: (_val, record) => (
          <Link
            style={{ fontFamily: 'monospace', fontSize: 12 }}
            onClick={() => void navigate(`/device/detail/${record.sn}`)}
          >
            {record.sn}
          </Link>
        ),
      },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 100,
        group: 'common',
        render: (_val, record) => (
          <StatusIndicator
            status={record.connStatus === 'online' ? 'online' : 'offline'}
            text={record.connStatus === 'online' ? t('status.online') : t('status.offline')}
          />
        ),
      },
      {
        key: 'alarmLevel',
        title: t('device.alarmLevel'),
        dataIndex: 'alarmLevel',
        width: 100,
        group: 'common',
        render: (_val, record) => (
          <Tag color={SEVERITY_COLOR[record.alarmLevel] ?? 'default'}>
            {SEVERITY_LABEL[record.alarmLevel] ?? record.alarmLevel}
          </Tag>
        ),
      },
      { key: 'hostName', title: t('device.hostName'), dataIndex: 'hostName', width: 150, ellipsis: true, group: 'common' },
      {
        key: 'networkType',
        title: t('device.radioMode'),
        dataIndex: 'networkType',
        width: 100,
        group: 'common',
        render: (_val, record) => {
          const colorMap: Record<string, string> = { eNB: 'blue', gNB: 'green', GSM: 'orange' };
          return <Tag color={colorMap[record.networkType] ?? 'default'}>{record.networkType || '-'}</Tag>;
        },
      },
      { key: 'productType', title: t('device.productType'), dataIndex: 'productType', width: 110, group: 'common' },
      { key: 'deviceModel', title: t('device.model'), dataIndex: 'deviceModel', width: 120, ellipsis: true, group: 'common' },
      { key: 'softwareVersion', title: t('device.softwareVersion'), dataIndex: 'softwareVersion', width: 140, ellipsis: true, group: 'common' },
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true, copyable: true, group: 'common' },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 120, group: 'common' },
      { key: 'ipAddress', title: t('device.ipAddress'), dataIndex: 'ipAddress', width: 140, mono: true, copyable: true, group: 'common' },
      {
        key: 'onlineTime',
        title: t('device.onlineTime'),
        dataIndex: 'onlineTime',
        width: 165,
        group: 'common',
        render: (_val, record) => fmtTime(record.onlineTime),
      },
      {
        key: 'offlineTime',
        title: t('device.offlineTime'),
        dataIndex: 'offlineTime',
        width: 165,
        group: 'common',
        render: (_val, record) => fmtTime(record.offlineTime),
      },
      { key: 'productName', title: t('device.productName'), dataIndex: 'productName', width: 130, hidden: true, group: 'common' },
      { key: 'firmwareVersion', title: t('device.firmwareVersion'), dataIndex: 'firmwareVersion', width: 140, hidden: true, ellipsis: true, group: 'common' },
      {
        key: 'onlineDuration',
        title: t('device.onlineDuration'),
        dataIndex: 'onlineDuration',
        width: 120,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtDuration(record.onlineDuration),
      },
      { key: 'upTime', title: t('device.upTime'), dataIndex: 'upTime', width: 120, hidden: true, group: 'common' },
      {
        key: 'firstOnlineTime',
        title: t('device.firstOnlineTime'),
        dataIndex: 'firstOnlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.firstOnlineTime),
      },
      {
        key: 'lastInformTime',
        title: t('device.lastInformTime'),
        dataIndex: 'lastInformTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.lastInformTime),
      },
      {
        key: 'lastOnlineTime',
        title: t('device.lastOnline'),
        dataIndex: 'lastOnlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.lastOnlineTime),
      },
      { key: 'siteName', title: t('device.siteName'), dataIndex: 'siteName', width: 130, hidden: true, ellipsis: true, group: 'common' },
      {
        key: 'opState',
        title: t('device.opState'),
        dataIndex: 'opState',
        width: 100,
        group: 'common',
        render: (_val, record) => {
          const colorMap: Record<string, string> = { active: 'success', inactive: 'error', unknown: 'default' };
          return <Tag color={colorMap[record.opState] ?? 'default'}>{record.opState}</Tag>;
        },
      },
      { key: 'ueCount', title: t('device.ueCount'), dataIndex: 'ueCount', width: 80, group: 'common' },
      { key: 'rfStatus', title: t('device.rfStatus'), dataIndex: 'rfStatus', width: 110, hidden: true, group: 'common' },
      { key: 'longitude', title: t('device.longitude'), dataIndex: 'longitude', width: 110, hidden: true, group: 'common' },
      { key: 'latitude', title: t('device.latitude'), dataIndex: 'latitude', width: 110, hidden: true, group: 'common' },
      { key: 'gpsHeight', title: t('device.gpsHeight'), dataIndex: 'gpsHeight', width: 100, hidden: true, group: 'common' },
      { key: 'gpsSatelliteCount', title: t('device.gpsSatelliteCount'), dataIndex: 'gpsSatelliteCount', width: 110, hidden: true, group: 'common' },
      { key: 'installAddress', title: t('device.installAddress'), dataIndex: 'installAddress', width: 180, hidden: true, ellipsis: true, group: 'common' },

      // =====================================================================
      // eNB 字段 — LTE 独有或 LTE+GSM 共有
      // =====================================================================
      { key: 'enbId', title: 'eNodeB ID', dataIndex: 'enbId', width: 110, hidden: true, group: 'eNB' },
      { key: 'cellId', title: t('device.cellId'), dataIndex: 'cellId', width: 80, hidden: true, group: 'eNB' },
      { key: 'eci', title: 'ECI', dataIndex: 'eci', width: 120, hidden: true, group: 'eNB' },
      { key: 'pci', title: 'PCI', dataIndex: 'pci', width: 80, hidden: true, group: 'eNB' },
      { key: 'plmnId', title: 'PLMN', dataIndex: 'plmnId', width: 90, hidden: true, group: 'eNB' },
      { key: 'tac', title: 'TAC', dataIndex: 'tac', width: 80, hidden: true, group: 'eNB' },
      { key: 'subframeAssignment', title: t('device.subframeAssignment'), dataIndex: 'subframeAssignment', width: 100, hidden: true, group: 'eNB' },
      { key: 'specialSubframe', title: t('device.specialSubframe'), dataIndex: 'specialSubframe', width: 120, hidden: true, group: 'eNB' },
      { key: 'rootIndex', title: t('device.rootIndex'), dataIndex: 'rootIndex', width: 110, hidden: true, group: 'eNB' },
      { key: 'siteId', title: 'Site ID', dataIndex: 'siteId', width: 130, hidden: true, group: 'eNB' },
      { key: 'bandwidth', title: t('device.bandwidth'), dataIndex: 'bandwidth', width: 90, hidden: true, group: 'eNB' },
      { key: 'dlEarfcn', title: t('device.dlEarfcn'), dataIndex: 'dlEarfcn', width: 110, hidden: true, group: 'eNB' },
      { key: 'ulEarfcn', title: t('device.ulEarfcn'), dataIndex: 'ulEarfcn', width: 110, hidden: true, group: 'eNB' },
      { key: 'networkModel', title: t('device.networkModel'), dataIndex: 'networkModel', width: 110, hidden: true, group: 'eNB' },
      { key: 'txPower', title: 'Tx Power', dataIndex: 'txPower', width: 100, hidden: true, group: 'eNB' },
      { key: 'band', title: 'Band', dataIndex: 'band', width: 90, hidden: true, group: 'eNB' },
      { key: 'mmeStatus', title: t('device.mmeStatus'), dataIndex: 'mmeStatus', width: 110, hidden: true, group: 'eNB' },
      { key: 'pmReportStatus', title: t('device.pmReportStatus'), dataIndex: 'pmReportStatus', width: 120, hidden: true, group: 'eNB' },
      { key: 'cpeCount', title: t('device.cpeCount'), dataIndex: 'cpeCount', width: 100, hidden: true, group: 'eNB' },
      { key: 'gpsVersion', title: t('device.gpsVersion'), dataIndex: 'gpsVersion', width: 100, hidden: true, group: 'eNB' },
      { key: 'rom', title: 'ROM', dataIndex: 'rom', width: 100, hidden: true, group: 'eNB' },
      { key: 'remark', title: t('device.remark'), dataIndex: 'remark', width: 150, hidden: true, ellipsis: true, group: 'eNB' },
      { key: 'lockStatus', title: t('device.lockStatus'), dataIndex: 'lockStatus', width: 100, hidden: true, group: 'eNB' },
      { key: 'wanSpeed', title: t('device.wanSpeed'), dataIndex: 'wanSpeed', width: 110, hidden: true, group: 'eNB' },
      { key: 'serviceStatus', title: t('device.serviceStatus'), dataIndex: 'serviceStatus', width: 100, hidden: true, group: 'eNB' },
      { key: 'adminState', title: 'Admin State', dataIndex: 'adminState', width: 110, hidden: true, group: 'eNB' },
      { key: 'multiPlmnEnable', title: 'Multi PLMN', dataIndex: 'multiPlmnEnable', width: 110, hidden: true, group: 'eNB' },
      { key: 'ipsecAddr', title: t('device.ipsecAddr'), dataIndex: 'ipsecAddr', width: 140, hidden: true, mono: true, group: 'eNB' },
      { key: 'mmepoolIpsecAddr', title: t('device.mmepoolIpsecAddr'), dataIndex: 'mmepoolIpsecAddr', width: 160, hidden: true, mono: true, group: 'eNB' },
      { key: 'mechanicalDowntilt', title: t('device.mechanicalDowntilt'), dataIndex: 'mechanicalDowntilt', width: 110, hidden: true, group: 'eNB' },
      { key: 'electronicDowntilt', title: t('device.electronicDowntilt'), dataIndex: 'electronicDowntilt', width: 110, hidden: true, group: 'eNB' },
      { key: 'verticalBeamWidth', title: t('device.verticalBeamWidth'), dataIndex: 'verticalBeamWidth', width: 120, hidden: true, group: 'eNB' },
      { key: 'horizontalAzimuth', title: t('device.horizontalAzimuth'), dataIndex: 'horizontalAzimuth', width: 120, hidden: true, group: 'eNB' },

      // =====================================================================
      // gNB 字段 — 5G NR 独有
      // =====================================================================
      { key: 'gnbId', title: 'gNB ID', dataIndex: 'gnbId', width: 110, hidden: true, group: 'gNB' },
      { key: 'nrCellId', title: 'NR Cell ID', dataIndex: 'nrCellId', width: 120, hidden: true, group: 'gNB' },
      { key: 'amfStatus', title: t('device.amfStatus'), dataIndex: 'amfStatus', width: 110, hidden: true, group: 'gNB' },
      {
        key: 'halobFlag',
        title: 'HaloB',
        dataIndex: 'halobFlag',
        width: 80,
        hidden: true,
        group: 'gNB',
        render: (_val, record) => (record.halobFlag ? t('common.yes') : t('common.no')),
      },
      { key: 'syncStatus', title: t('device.syncStatus'), dataIndex: 'syncStatus', width: 100, hidden: true, group: 'gNB' },
      { key: 'validity', title: t('device.validity'), dataIndex: 'validity', width: 120, hidden: true, group: 'gNB' },
      { key: 'euCount', title: t('device.euCount'), dataIndex: 'euCount', width: 80, hidden: true, group: 'gNB' },
      { key: 'ruCount', title: t('device.ruCount'), dataIndex: 'ruCount', width: 80, hidden: true, group: 'gNB' },
      { key: 'rollbackVersion', title: t('device.rollbackVersion'), dataIndex: 'rollbackVersion', width: 140, hidden: true, group: 'gNB' },
      { key: 'sasParam', title: t('device.sasParam'), dataIndex: 'sasParam', width: 120, hidden: true, group: 'gNB' },
      { key: 'euRu', title: t('device.euRu'), dataIndex: 'euRu', width: 90, hidden: true, group: 'gNB' },
      { key: 'halobLicense', title: t('device.halobLicense'), dataIndex: 'halobLicense', width: 120, hidden: true, group: 'gNB' },
      { key: 'energySaving', title: t('device.energySaving'), dataIndex: 'energySaving', width: 100, hidden: true, group: 'gNB' },
      { key: 'gnbTopoCellmgr', title: t('device.gnbTopoCellmgr'), dataIndex: 'gnbTopoCellmgr', width: 120, hidden: true, group: 'gNB' },
      { key: 'sslCertValidity', title: t('device.sslCertValidity'), dataIndex: 'sslCertValidity', width: 140, hidden: true, group: 'gNB' },

      // =====================================================================
      // GSM 字段 — GSM 独有
      // =====================================================================
      { key: 'lac', title: 'LAC', dataIndex: 'lac', width: 80, hidden: true, group: 'GSM' },
      { key: 'arfcn', title: t('device.arfcn'), dataIndex: 'arfcn', width: 90, hidden: true, group: 'GSM' },
      { key: 'uplinkFrequency', title: t('device.uplinkFrequency'), dataIndex: 'uplinkFrequency', width: 120, hidden: true, group: 'GSM' },
      { key: 'downlinkFrequency', title: t('device.downlinkFrequency'), dataIndex: 'downlinkFrequency', width: 120, hidden: true, group: 'GSM' },
      { key: 'bscLinkStatus', title: t('device.bscLinkStatus'), dataIndex: 'bscLinkStatus', width: 120, hidden: true, group: 'GSM' },
      { key: 'bscSelect', title: 'BSC Select', dataIndex: 'bscSelect', width: 100, hidden: true, group: 'GSM' },
      { key: 'bscSerialNumber', title: t('device.bscSerialNumber'), dataIndex: 'bscSerialNumber', width: 150, hidden: true, group: 'GSM' },
      { key: 'btsNum', title: t('device.btsNum'), dataIndex: 'btsNum', width: 80, hidden: true, group: 'GSM' },
      { key: 'ipaUnitId', title: 'IPA Unit ID', dataIndex: 'ipaUnitId', width: 120, hidden: true, group: 'GSM' },
      { key: 'omlRemoteIp', title: 'OML Remote IP', dataIndex: 'omlRemoteIp', width: 140, hidden: true, mono: true, group: 'GSM' },
      { key: 'omlRemoteIpBak', title: 'OML Remote IP Bak', dataIndex: 'omlRemoteIpBak', width: 160, hidden: true, mono: true, group: 'GSM' },

      // =====================================================================
      // 操作列
      // =====================================================================
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 120,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => void navigate(`/device/detail/${record.sn}`)}
            >
              {t('common.detail')}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => void navigate(`/device/edit/${record.id}`)}
            >
              {t('common.edit')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete([record.id])}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
    ],
    [navigate, handleDelete, t, fmtTime, fmtDuration, SEVERITY_LABEL]
  );

  const batchActions = useMemo(
    (): BatchAction[] => [
      {
        key: 'batch-delete',
        label: t('common.batchDelete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: (keys) => handleDelete(keys as string[]),
      },
      {
        key: 'batch-config',
        label: t('common.batchConfig'),
        icon: <SettingOutlined />,
        onClick: () => void message.info(t('common.featureInDev')),
      },
      {
        key: 'export',
        label: t('common.export'),
        icon: <DownloadOutlined />,
        onClick: () => void message.info(t('common.exportInProgress')),
      },
    ],
    [handleDelete, t]
  );

  return (
    <ListPageLayout
      title={t('nav.device.list')}
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => void navigate('/device/registration')}
        >
          {t('common.addDevice')}
        </Button>
      }
    >
      <FilterBar
        filterId="device-list"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <StatisticsPanel items={statsItems} style={{ marginBottom: 8 }} />

      <DataTable<Device>
        tableId="device-list-table"
        columns={columns}
        dataSource={devices}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedRowKeys}
        onSelectionChange={(keys) => setSelectedRowKeys(keys)}
        total={total}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={(page, size) => {
          setCurrentPage(page);
          setPageSize(size);
        }}
        batchActions={batchActions}
        onRefresh={() => void refetch()}
        defaultDensity="compact"
      />
    </ListPageLayout>
  );
}
