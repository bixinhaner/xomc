import React, { useState, useMemo, useCallback } from 'react';
import { Tabs, Button, Space, DatePicker, Select, Input, Radio, Typography, Dropdown, Modal, Form, message, Switch, InputNumber, TimePicker, List, Spin, Segmented, Collapse } from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  DownloadOutlined,
  TableOutlined,
  BarChartOutlined,
  EditOutlined,
  DeleteOutlined,
  CopyOutlined,
  StarOutlined,
  StarFilled,
  SettingOutlined,
  ClockCircleOutlined,
  MoreOutlined,
  FileOutlined,
  TeamOutlined,
  UserOutlined,
  InfoCircleOutlined,
  ExportOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

const { RangePicker } = DatePicker;
const { Title, Text } = Typography;

// Mock template data
const PUBLIC_TEMPLATES: TemplateItem[] = [
  { id: 'tpl-1', name: 'eNB基础KPI', isDefault: true, reportSwitch: '0', isPublic: '1', isOneSelf: '0', isAdmin: '0' },
  { id: 'tpl-2', name: 'gNB性能指标', isDefault: false, reportSwitch: '1', isPublic: '1', isOneSelf: '0', isAdmin: '0' },
  { id: 'tpl-3', name: '小区吞吐量', isDefault: false, reportSwitch: '0', isPublic: '1', isOneSelf: '0', isAdmin: '0' },
  { id: 'tpl-6', name: '切换成功率分析', isDefault: false, reportSwitch: '0', isPublic: '1', isOneSelf: '1', isAdmin: '0' },
  { id: 'tpl-7', name: '用户数统计', isDefault: false, reportSwitch: '1', isPublic: '1', isOneSelf: '1', isAdmin: '1' },
];

const PRIVATE_TEMPLATES: TemplateItem[] = [
  { id: 'tpl-4', name: '我的eNB模板', isDefault: false, reportSwitch: '0', isPublic: '0', isOneSelf: '1', creator: 'admin', isAdmin: '0' },
  { id: 'tpl-5', name: '测试模板', isDefault: false, reportSwitch: '0', isPublic: '0', isOneSelf: '1', creator: 'admin', isAdmin: '0' },
  { id: 'tpl-8', name: '自定义报表', isDefault: false, reportSwitch: '1', isPublic: '0', isOneSelf: '1', creator: 'admin', isAdmin: '0' },
  { id: 'tpl-9', name: '性能监控模板', isDefault: false, reportSwitch: '0', isPublic: '0', isOneSelf: '0', creator: 'zhangsan', isAdmin: '0' },
  { id: 'tpl-10', name: '告警分析', isDefault: false, reportSwitch: '1', isPublic: '0', isOneSelf: '0', creator: 'zhangsan', isAdmin: '0' },
  { id: 'tpl-11', name: '容量规划', isDefault: false, reportSwitch: '0', isPublic: '0', isOneSelf: '0', creator: 'lisi', isAdmin: '0' },
];

// Mock KPI data
const MOCK_KPI_DATA = [
  { key: '1', serialNumber: 'ENB00001', hostName: '北京朝阳基站01', enodeId: '100001', cellId: '1', eci: '100001001', groupName: '北京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.2%', ERAB_SR: '98.5%', DL_THP: '145.6', UL_THP: '32.1' },
  { key: '2', serialNumber: 'ENB00002', hostName: '北京海淀基站01', enodeId: '100002', cellId: '1', eci: '100002001', groupName: '北京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.9%', ERAB_SR: '97.8%', DL_THP: '132.1', UL_THP: '28.5' },
  { key: '3', serialNumber: 'GNB00001', hostName: '北京5G基站01', enodeId: '200001', cellId: '1', eci: '200001001', groupName: '北京5G区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.5%', ERAB_SR: '99.1%', DL_THP: '856.2', UL_THP: '125.3' },
];

interface TemplateItem {
  id: string;
  name: string;
  isDefault?: boolean;
  reportSwitch?: string;
  isPublic: string;
  isOneSelf?: string;
  isAdmin?: string;
  creator?: string;
}

export default function KPIQuery() {
  const t = useT();
  const token = useThemeToken();
  const [searchText, setSearchText] = useState('');
  const [templateTab, setTemplateTab] = useState<'public' | 'private'>('public');
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>('tpl-1');
  const [openTabs, setOpenTabs] = useState<string[]>(['tpl-1']);
  const [activeTab, setActiveTab] = useState<string>('tpl-1');
  const [viewMode, setViewMode] = useState<'table' | 'chart'>('table');
  const [granularity, setGranularity] = useState('15');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [deviceType, setDeviceType] = useState<'1' | '2'>('2');
  const [deviceSearch, setDeviceSearch] = useState('');
  const [loading, setLoading] = useState(false);

  // Template dialog state
  const [templateDialogOpen, setTemplateDialogOpen] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TemplateItem | null>(null);
  const [templateForm] = Form.useForm();

  // Report config drawer state
  const [reportDrawerOpen, setReportDrawerOpen] = useState(false);
  const [reportForm] = Form.useForm();

  // All templates combined
  const allTemplates = useMemo(() => [...PUBLIC_TEMPLATES, ...PRIVATE_TEMPLATES], []);

  // Time granularity options with i18n
  const granularityOptions = useMemo(() => [
    { label: t('perf.query.granularity15min'), value: '15' },
    { label: t('perf.query.granularity60min'), value: '60' },
    { label: t('perf.query.granularity24hour'), value: '1440' },
    { label: t('perf.query.granularityWeek'), value: '10080' },
    { label: t('perf.query.granularityMonth'), value: '43200' },
  ], [t]);

  // Get template menu items - all templates have the same menu items
  const getTemplateMenuItems = useCallback((template: TemplateItem): MenuProps['items'] => {
    const items: MenuProps['items'] = [];

    // 已为默认 / 设为默认
    if (template.isDefault) {
      items.push({
        key: 'isDefault',
        label: t('perf.query.isDefault'),
        icon: <StarFilled style={{ color: '#faad14' }} />,
        disabled: true,
      });
    } else {
      items.push({
        key: 'setDefault',
        label: t('perf.query.setAsDefault'),
        icon: <StarOutlined />,
      });
    }

    // 详情
    items.push({
      key: 'detail',
      label: t('common.detail'),
      icon: <InfoCircleOutlined />,
    });

    // 修改
    items.push({
      key: 'edit',
      label: t('common.edit'),
      icon: <EditOutlined />,
    });

    // 导出指标
    items.push({
      key: 'exportKpi',
      label: t('perf.query.exportKpi'),
      icon: <ExportOutlined />,
    });

    // 定时报表
    items.push({
      key: 'report',
      label: t('perf.query.reportConfig'),
      icon: <SettingOutlined />,
    });

    // 删除 - 非默认模板才显示
    if (!template.isDefault) {
      items.push({ type: 'divider' });
      items.push({
        key: 'delete',
        label: t('common.delete'),
        icon: <DeleteOutlined />,
        danger: true,
      });
    }

    // 复制模板
    items.push({
      key: 'copy',
      label: t('common.copy'),
      icon: <CopyOutlined />,
    });

    return items;
  }, [t]);

  // Handle template menu click
  const handleTemplateMenuClick = useCallback((key: string, template: TemplateItem) => {
    switch (key) {
      case 'detail':
        // TODO: Show template detail modal
        Modal.info({
          title: t('perf.query.templateDetail'),
          content: (
            <div>
              <p><strong>{t('perf.query.templateName')}:</strong> {template.name}</p>
              <p><strong>{t('perf.query.templateType')}:</strong> {template.isPublic === '1' ? t('perf.query.publicTemplate') : t('perf.query.privateTemplate')}</p>
              {template.creator && <p><strong>{t('perf.query.creator')}:</strong> {template.creator}</p>}
            </div>
          ),
        });
        break;
      case 'edit':
        setEditingTemplate(template);
        templateForm.setFieldsValue({ name: template.name, isPublic: template.isPublic });
        setTemplateDialogOpen(true);
        break;
      case 'copy':
        void message.success(t('common.success'));
        break;
      case 'setDefault':
        Modal.confirm({
          title: t('perf.query.setAsDefault'),
          content: t('perf.query.setAsDefaultConfirm', { name: template.name }),
          onOk: () => void message.success(t('common.success')),
        });
        break;
      case 'exportKpi':
        void message.success(t('common.exportInProgress'));
        break;
      case 'report':
        setReportDrawerOpen(true);
        break;
      case 'delete':
        Modal.confirm({
          title: t('common.confirmDelete'),
          content: t('common.deleteConfirmMsg'),
          okType: 'danger',
          onOk: () => void message.success(t('common.deleteSuccess')),
        });
        break;
    }
  }, [templateForm, t]);

  // Filter templates by search text
  const filteredPublicTemplates = useMemo(() => {
    if (!searchText.trim()) return PUBLIC_TEMPLATES;
    return PUBLIC_TEMPLATES.filter((tpl) =>
      tpl.name.toLowerCase().includes(searchText.toLowerCase())
    );
  }, [searchText]);

  const filteredPrivateTemplates = useMemo(() => {
    if (!searchText.trim()) return PRIVATE_TEMPLATES;
    return PRIVATE_TEMPLATES.filter((tpl) =>
      tpl.name.toLowerCase().includes(searchText.toLowerCase())
    );
  }, [searchText]);

  // Group private templates by creator
  const groupedPrivateTemplates = useMemo(() => {
    const groups: Record<string, TemplateItem[]> = {};
    filteredPrivateTemplates.forEach((tpl) => {
      const creator = tpl.creator || 'unknown';
      if (!groups[creator]) {
        groups[creator] = [];
      }
      groups[creator].push(tpl);
    });
    return Object.entries(groups).map(([creator, templates]) => ({
      creator,
      templates,
      count: templates.length,
    }));
  }, [filteredPrivateTemplates]);

  // Handle template selection
  const handleTemplateSelect = useCallback((template: TemplateItem) => {
    setSelectedTemplateId(template.id);
    if (!openTabs.includes(template.id)) {
      setOpenTabs([...openTabs, template.id]);
    }
    setActiveTab(template.id);
  }, [openTabs]);

  // Handle tab close
  const handleTabClose = useCallback((targetKey: string) => {
    const newTabs = openTabs.filter((tab) => tab !== targetKey);
    setOpenTabs(newTabs);
    if (activeTab === targetKey && newTabs.length > 0) {
      setActiveTab(newTabs[newTabs.length - 1]);
    }
  }, [openTabs, activeTab]);

  // Handle query
  const handleQuery = useCallback(() => {
    setLoading(true);
    setTimeout(() => {
      setLoading(false);
      void message.success(t('common.success'));
    }, 1000);
  }, [t]);

  // Handle export
  const handleExport = useCallback((format: 'excel' | 'csv') => {
    void message.success(`${format.toUpperCase()} ${t('common.exportInProgress')}`);
  }, [t]);

  // Get template name by id
  const getTemplateName = useCallback((id: string): string => {
    const found = allTemplates.find((t) => t.id === id);
    return found?.name ?? id;
  }, [allTemplates]);

  // Render template item
  const renderTemplateItem = useCallback((template: TemplateItem) => (
    <List.Item
      key={template.id}
      onClick={() => handleTemplateSelect(template)}
      style={{
        padding: '8px 12px',
        cursor: 'pointer',
        background: selectedTemplateId === template.id ? token.colorPrimaryBg : 'transparent',
        borderRadius: 4,
        marginBottom: 4,
        transition: 'background 0.2s',
      }}
      onMouseEnter={(e) => {
        if (selectedTemplateId !== template.id) {
          e.currentTarget.style.background = token.colorBgTextHover;
        }
      }}
      onMouseLeave={(e) => {
        if (selectedTemplateId !== template.id) {
          e.currentTarget.style.background = 'transparent';
        }
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, overflow: 'hidden' }}>
          <FileOutlined style={{ color: '#8c8c8c', flexShrink: 0 }} />
          <Text ellipsis style={{ flex: 1 }}>{template.name}</Text>
          {template.isDefault && <StarOutlined style={{ color: '#faad14', fontSize: 12, flexShrink: 0 }} />}
          {template.reportSwitch === '1' && <ClockCircleOutlined style={{ color: '#52c41a', fontSize: 12, flexShrink: 0 }} />}
        </div>
        <Dropdown
          menu={{
            items: getTemplateMenuItems(template),
            onClick: (info) => {
              info.domEvent.stopPropagation();
              handleTemplateMenuClick(info.key, template);
            },
          }}
          trigger={['click']}
        >
          <Button
            type="text"
            size="small"
            icon={<MoreOutlined />}
            onClick={(e) => e.stopPropagation()}
            style={{ padding: '0 4px', flexShrink: 0 }}
          />
        </Dropdown>
      </div>
    </List.Item>
  ), [selectedTemplateId, token, handleTemplateSelect, handleTemplateMenuClick, getTemplateMenuItems]);

  // Table columns
  const columns: DataTableColumn<Record<string, unknown>>[] = useMemo(() => [
    { key: 'serialNumber', title: t('perf.query.serialNumber'), dataIndex: 'serialNumber', width: 160, fixed: 'left', mono: true },
    { key: 'hostName', title: t('perf.query.hostName'), dataIndex: 'hostName', width: 180, fixed: 'left' },
    { key: 'subStationName', title: t('perf.query.subStationName'), dataIndex: 'subStationName', width: 180 },
    { key: 'enodeId', title: t('perf.query.enodeId'), dataIndex: 'enodeId', width: 120, mono: true },
    { key: 'cellId', title: t('perf.query.cellId'), dataIndex: 'cellId', width: 100 },
    { key: 'eci', title: t('perf.query.eci'), dataIndex: 'eci', width: 140, mono: true },
    { key: 'groupName', title: t('perf.query.groupName'), dataIndex: 'groupName', width: 160 },
    { key: 'timeLevel', title: t('perf.query.timeLevel'), dataIndex: 'timeLevel', width: 100 },
    { key: 'startTime', title: t('perf.query.startTime'), dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: t('perf.query.endTime'), dataIndex: 'endTime', width: 160 },
    { key: 'RRC_SR', title: `${t('kpi.rrcSetupSuccessRate')}(%)`, dataIndex: 'RRC_SR', width: 140 },
    { key: 'ERAB_SR', title: `${t('kpi.erabSetupSuccessRate')}(%)`, dataIndex: 'ERAB_SR', width: 150 },
    { key: 'DL_THP', title: `${t('kpi.dlThroughput')}(Mbps)`, dataIndex: 'DL_THP', width: 140 },
    { key: 'UL_THP', title: `${t('kpi.ulThroughput')}(Mbps)`, dataIndex: 'UL_THP', width: 140 },
  ], [t]);

  // Left panel with tabs
  const leftPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header */}
      <div
        style={{
          padding: '12px 12px 8px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        <Title level={5} style={{ margin: 0, fontSize: 14 }}>
          {t('perf.query.templateTitle')}
        </Title>
        <Button
          type="text"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingTemplate(null);
            templateForm.resetFields();
            setTemplateDialogOpen(true);
          }}
        >
          {t('common.add')}
        </Button>
      </div>

      {/* Search */}
      <div style={{ padding: '8px 12px', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
        <Input
          placeholder={t('perf.query.searchTemplate')}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
          size="small"
        />
      </div>

      {/* Tabs for Public/Private */}
      <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
        {/* Segmented Control */}
        <div style={{
          padding: '12px 12px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
        }}>
          <Segmented
            value={templateTab}
            onChange={(value) => setTemplateTab(value as 'public' | 'private')}
            block
            style={{ width: '100%' }}
            options={[
              {
                value: 'public',
                label: (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '2px 0' }}>
                    <TeamOutlined />
                    <span>{t('perf.query.publicTemplate')}</span>
                    <span style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      minWidth: 18,
                      height: 18,
                      padding: '0 4px',
                      borderRadius: 9,
                      fontSize: 11,
                      backgroundColor: 'rgba(0,0,0,0.15)',
                    }}>
                      {filteredPublicTemplates.length}
                    </span>
                  </div>
                ),
              },
              {
                value: 'private',
                label: (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '2px 0' }}>
                    <UserOutlined />
                    <span>{t('perf.query.privateTemplate')}</span>
                    <span style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      minWidth: 18,
                      height: 18,
                      padding: '0 4px',
                      borderRadius: 9,
                      fontSize: 11,
                      backgroundColor: 'rgba(0,0,0,0.15)',
                    }}>
                      {filteredPrivateTemplates.length}
                    </span>
                  </div>
                ),
              },
            ]}
          />
        </div>

        {/* Tab Content */}
        <div style={{ flex: 1, overflow: 'auto', padding: '8px 8px' }}>
          {templateTab === 'public' ? (
            filteredPublicTemplates.length > 0 ? (
              filteredPublicTemplates.map(renderTemplateItem)
            ) : (
              <div style={{ textAlign: 'center', padding: 24, color: token.colorTextSecondary }}>
                {t('common.noData')}
              </div>
            )
          ) : (
            groupedPrivateTemplates.length > 0 ? (
              <Collapse
                defaultActiveKey={groupedPrivateTemplates[0]?.creator}
                ghost
                expandIconPosition="end"
                style={{ background: 'transparent' }}
                items={groupedPrivateTemplates.map((group) => ({
                  key: group.creator,
                  label: (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%' }}>
                      <UserOutlined style={{ color: token.colorPrimary }} />
                      <span style={{ flex: 1 }}>{group.creator}</span>
                      <span style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        minWidth: 18,
                        height: 18,
                        padding: '0 4px',
                        borderRadius: 9,
                        fontSize: 11,
                        backgroundColor: token.colorPrimaryBg,
                        color: token.colorPrimary,
                      }}>
                        {group.count}
                      </span>
                    </div>
                  ),
                  children: (
                    <div style={{ paddingLeft: 24 }}>
                      {group.templates.map((template) => renderTemplateItem(template))}
                    </div>
                  ),
                }))}
              />
            ) : (
              <div style={{ textAlign: 'center', padding: 24, color: token.colorTextSecondary }}>
                {t('common.noData')}
              </div>
            )
          )}
        </div>
      </div>
    </div>
  );

  // Right content panel
  const renderRightPanel = (tabId: string) => {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        {/* Query conditions */}
        <div
          style={{
            padding: '12px 16px',
            background: token.colorBgContainer,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, alignItems: 'center' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text style={{ whiteSpace: 'nowrap' }}>{t('perf.query.queryObjectType')}</Text>
              <Radio.Group value={deviceType} onChange={(e) => setDeviceType(e.target.value)} size="small">
                <Radio.Button value="2">{t('device.name')}</Radio.Button>
                <Radio.Button value="1">{t('device.group')}</Radio.Button>
              </Radio.Group>
            </div>
            <Input
              placeholder={deviceType === '2' ? t('perf.query.deviceSearchPlaceholder') : t('perf.query.groupSearchPlaceholder')}
              prefix={<SearchOutlined />}
              value={deviceSearch}
              onChange={(e) => setDeviceSearch(e.target.value)}
              allowClear
              style={{ width: 200 }}
              size="small"
            />
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text style={{ whiteSpace: 'nowrap' }}>{t('perf.query.granularity')}</Text>
              <Select
                value={granularity}
                onChange={setGranularity}
                options={granularityOptions}
                style={{ width: 100 }}
                size="small"
              />
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text style={{ whiteSpace: 'nowrap' }}>{t('perf.query.timeRange')}</Text>
              <RangePicker
                value={dateRange}
                onChange={(dates) => setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs] | null)}
                showTime={{ format: 'HH:mm' }}
                format="YYYY-MM-DD HH:mm"
                style={{ width: 340 }}
                size="small"
              />
            </div>
            <Space size="small">
              <Button type="primary" onClick={handleQuery} loading={loading} size="small">
                {t('common.query')}
              </Button>
              <Dropdown
                menu={{
                  items: [
                    { key: 'excel', label: t('perf.query.exportExcel'), onClick: () => handleExport('excel') },
                    { key: 'csv', label: t('perf.query.exportCsv'), onClick: () => handleExport('csv') },
                  ],
                }}
              >
                <Button icon={<DownloadOutlined />} size="small">{t('common.export')}</Button>
              </Dropdown>
            </Space>
          </div>
        </div>

        {/* View mode toggle */}
        <div style={{ padding: '8px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
          <Text strong>{getTemplateName(tabId)}</Text>
          <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} size="small">
            <Radio.Button value="table"><TableOutlined /> {t('perf.query.tableView')}</Radio.Button>
            <Radio.Button value="chart"><BarChartOutlined /> {t('perf.query.chartView')}</Radio.Button>
          </Radio.Group>
        </div>

        {/* Result area */}
        <div style={{ flex: 1, overflow: 'hidden', padding: 16 }}>
          {viewMode === 'table' ? (
            <Spin spinning={loading}>
              <DataTable
                tableId={`kpi-query-${tabId}`}
                columns={columns}
                dataSource={MOCK_KPI_DATA}
                rowKey="key"
                scroll={{ x: 1800, y: 'calc(100% - 56px)' }}
                showPagination
                pageSize={20}
              />
            </Spin>
          ) : (
            <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
              <div style={{ marginBottom: 16, display: 'flex', gap: 8 }}>
                <Select
                  placeholder={t('perf.query.selectDevice')}
                  style={{ width: 200 }}
                  size="small"
                  options={[
                    { value: 'ENB00001', label: '北京朝阳基站01' },
                    { value: 'ENB00002', label: '北京海淀基站01' },
                    { value: 'GNB00001', label: '北京5G基站01' },
                  ]}
                />
                <Select
                  placeholder={t('perf.query.selectKpi')}
                  style={{ width: 200 }}
                  mode="multiple"
                  size="small"
                  options={[
                    { value: 'RRC_SR', label: t('kpi.rrcSetupSuccessRate') },
                    { value: 'ERAB_SR', label: t('kpi.erabSetupSuccessRate') },
                    { value: 'DL_THP', label: t('kpi.dlThroughput') },
                  ]}
                />
                <Button type="primary" size="small">{t('common.query')}</Button>
              </div>
              <div style={{ flex: 1, minHeight: 0 }}>
                <LineChart
                  series={[
                    { name: t('kpi.rrcSetupSuccessRate'), data: [99.2, 99.5, 98.8, 99.1, 99.3, 99.0, 98.9] },
                    { name: t('kpi.erabSetupSuccessRate'), data: [98.5, 98.8, 98.2, 98.6, 98.9, 98.4, 98.7] },
                  ]}
                  xData={['10:00', '10:15', '10:30', '10:45', '11:00', '11:15', '11:30']}
                  height={400}
                />
              </div>
            </div>
          )}
        </div>
      </div>
    );
  };

  // Tab items
  const tabItems = useMemo(() => {
    return openTabs.map((tabId) => ({
      key: tabId,
      label: getTemplateName(tabId),
      closable: openTabs.length > 1,
      children: renderRightPanel(tabId),
    }));
  }, [openTabs, getTemplateName, renderRightPanel]);

  return (
    <>
      <TreeListPageLayout tree={leftPanel} defaultTreeWidth={280}>
        <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
          {openTabs.length > 0 ? (
            <Tabs
              type="editable-card"
              activeKey={activeTab}
              onChange={setActiveTab}
              onEdit={(targetKey, action) => {
                if (action === 'remove' && typeof targetKey === 'string') {
                  handleTabClose(targetKey);
                }
              }}
              items={tabItems}
              style={{ height: '100%' }}
              tabBarStyle={{ margin: 0, padding: '0 16px', background: token.colorBgContainer }}
            />
          ) : (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: token.colorTextSecondary }}>
              <div style={{ textAlign: 'center' }}>
                <FileOutlined style={{ fontSize: 48, marginBottom: 16, opacity: 0.3 }} />
                <div>{t('perf.query.selectTemplate')}</div>
              </div>
            </div>
          )}
        </div>
      </TreeListPageLayout>

      {/* Template dialog */}
      <Modal
        title={editingTemplate ? t('perf.query.editTemplate') : t('perf.query.addTemplate')}
        open={templateDialogOpen}
        onCancel={() => setTemplateDialogOpen(false)}
        onOk={() => {
          templateForm.validateFields().then(() => {
            void message.success(t('common.success'));
            setTemplateDialogOpen(false);
          }).catch(() => {});
        }}
        width={500}
      >
        <Form form={templateForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label={t('perf.query.templateName')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <Input placeholder={t('perf.query.templateNamePlaceholder')} />
          </Form.Item>
          <Form.Item
            name="isPublic"
            label={t('perf.query.templateType')}
            rules={[{ required: true }]}
            initialValue="0"
          >
            <Radio.Group>
              <Radio value="1">{t('perf.query.publicTemplate')}</Radio>
              <Radio value="0">{t('perf.query.privateTemplate')}</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={3} placeholder={t('common.pleaseInput')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Report config drawer */}
      <Modal
        title={t('perf.query.reportConfig')}
        open={reportDrawerOpen}
        onCancel={() => setReportDrawerOpen(false)}
        onOk={() => {
          reportForm.validateFields().then(() => {
            void message.success(t('common.success'));
            setReportDrawerOpen(false);
          }).catch(() => {});
        }}
        width={600}
      >
        <Form form={reportForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('perf.query.enableReport')} name="reportStatus" valuePropName="checked" initialValue={false}>
            <Switch />
          </Form.Item>
          <Form.Item label={t('perf.query.reportPeriod')} name="reportPeriod">
            <Select mode="multiple" placeholder={t('common.pleaseSelect')} options={granularityOptions} />
          </Form.Item>
          <Form.Item label={t('perf.query.reportTime')} name="reportTime">
            <TimePicker format="HH:mm" style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('perf.query.enableEmail')} name="mailStatus" valuePropName="checked" initialValue={false}>
            <Switch />
          </Form.Item>
          <Form.Item label={t('perf.query.emailAddress')} name="mailAddress">
            <Input placeholder={t('perf.query.emailPlaceholder')} />
          </Form.Item>
          <Form.Item label={t('perf.query.enableFtp')} name="ftpSwitch" valuePropName="checked" initialValue={false}>
            <Switch />
          </Form.Item>
          <Form.Item label={t('perf.query.ftpProtocol')} name="ftpProtocol">
            <Radio.Group>
              <Radio value="sftp">SFTP</Radio>
              <Radio value="ftp">FTP</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item label={t('perf.query.ftpPath')} name="ftpPath">
            <Input placeholder="/data/reports" />
          </Form.Item>
          <Form.Item label={t('perf.query.ftpIp')} name="ftpIp">
            <Input placeholder="192.168.1.100" />
          </Form.Item>
          <Form.Item label={t('perf.query.ftpPort')} name="ftpPort">
            <InputNumber min={1} max={65535} placeholder="22" style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('perf.query.ftpUser')} name="ftpUser">
            <Input placeholder="admin" />
          </Form.Item>
          <Form.Item label={t('perf.query.ftpPassword')} name="ftpPassword">
            <Input.Password placeholder="******" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
