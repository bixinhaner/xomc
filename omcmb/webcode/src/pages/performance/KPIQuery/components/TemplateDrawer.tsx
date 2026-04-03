import React, { useState, useMemo } from 'react';
import {
  Drawer,
  Form,
  Input,
  Radio,
  Checkbox,
  Button,
  Space,
  Card,
  Select,
  App,
  Row,
  Col,
  Tree,
  Typography,
  Tooltip,
  Empty,
} from 'antd';
import {
  InfoCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { DataNode, TreeProps } from 'antd/es/tree';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

const { Text } = Typography;
const { TextArea } = Input;

type DrawerMode = 'add' | 'edit' | 'view' | 'copy';

interface TemplateDrawerProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: TemplateFormValues) => void;
  loading?: boolean;
  mode?: DrawerMode;
  initialValues?: Partial<TemplateFormValues>;
  networkType?: 'enb' | 'gnb' | 'egw';
}

export interface TemplateFormValues {
  // 基本信息
  tempName: string;
  isPublic: '0' | '1';
  description: string;
  creator?: string;
  updator?: string;
  // 周期设定
  reportPeriod: '15' | '60' | '1440';
  checkAll: '0' | '1';
  week: string[];
  hour: string[];
  busyTime: '' | '6' | '8';
  // 设备选择
  selDeviceType: '1' | '2';
  selectedDevices: string[];
  selectedGroups: string[];
  // 指标选择
  indicatorLevel?: 'device' | 'plmn';
  selectedKpis: string[];
  // 设为默认
  isDefault: boolean;
}

// 小时选择选项
const HOUR_OPTIONS = Array.from({ length: 24 }, (_, i) => ({
  label: `${i}:00`,
  value: String(i),
}));

// 忙时预设
const BUSY_HOUR_PRESETS: Record<string, string[]> = {
  '6': ['8', '9', '10', '18', '19', '20'],
  '8': ['8', '9', '10', '11', '18', '19', '20', '21'],
};

// 默认表单值
const DEFAULT_FORM_VALUES: TemplateFormValues = {
  tempName: '',
  isPublic: '0',
  description: '',
  reportPeriod: '15',
  checkAll: '1',
  week: [],
  hour: [],
  busyTime: '',
  selDeviceType: '2',
  selectedDevices: [],
  selectedGroups: [],
  indicatorLevel: 'device',
  selectedKpis: [],
  isDefault: false,
};

// Mock 设备组数据 (示例数据，实际使用时从 API 获取)
const MOCK_DEVICE_GROUPS: DataNode[] = [
  { key: 'group-1', title: 'Beijing Area', children: [
    { key: 'group-1-1', title: 'Beijing Chaoyang' },
    { key: 'group-1-2', title: 'Beijing Haidian' },
  ]},
  { key: 'group-2', title: 'Shanghai Area', children: [
    { key: 'group-2-1', title: 'Shanghai Pudong' },
    { key: 'group-2-2', title: 'Shanghai Xuhui' },
  ]},
  { key: 'group-3', title: 'Guangzhou Area' },
  { key: 'group-4', title: 'Shenzhen Area' },
];

// Mock 设备数据 (示例数据)
const MOCK_DEVICES: DataNode[] = [
  { key: 'ENB00001', title: 'ENB00001 - Beijing Chaoyang BS01' },
  { key: 'ENB00002', title: 'ENB00002 - Beijing Haidian BS01' },
  { key: 'ENB00003', title: 'ENB00003 - Beijing Dongcheng BS01' },
  { key: 'GNB00001', title: 'GNB00001 - Beijing 5G BS01' },
  { key: 'GNB00002', title: 'GNB00002 - Shanghai 5G BS01' },
];

export default function TemplateDrawer({
  open,
  onClose,
  onSubmit,
  loading = false,
  mode = 'add',
  initialValues,
  networkType = 'enb',
}: TemplateDrawerProps) {
  const t = useT();
  const token = useThemeToken();
  const { message } = App.useApp();
  const [form] = Form.useForm();

  // 周选择选项 - 使用 i18n
  const WEEK_OPTIONS = [
    { label: t('perf.query.week.sun'), value: '7' },
    { label: t('perf.query.week.mon'), value: '1' },
    { label: t('perf.query.week.tue'), value: '2' },
    { label: t('perf.query.week.wed'), value: '3' },
    { label: t('perf.query.week.thu'), value: '4' },
    { label: t('perf.query.week.fri'), value: '5' },
    { label: t('perf.query.week.sat'), value: '6' },
  ];

  // Mock KPI 分类数据 - 使用 i18n
  const MOCK_KPI_CATEGORIES: DataNode[] = useMemo(() => [
    {
      title: t('perf.query.accessKpi'),
      key: 'cat-1',
      children: [
        { title: t('kpi.rrcSetupSuccessRate'), key: 'kpi-rrc-sr' },
        { title: t('kpi.erabSetupSuccessRate'), key: 'kpi-erab-sr' },
        { title: t('kpi.accessRate'), key: 'kpi-radio-sr' },
      ],
    },
    {
      title: t('perf.query.throughputKpi'),
      key: 'cat-2',
      children: [
        { title: t('kpi.dlThroughput'), key: 'kpi-dl-thp' },
        { title: t('kpi.ulThroughput'), key: 'kpi-ul-thp' },
        { title: t('kpi.prbUtilization'), key: 'kpi-dl-prb' },
        { title: t('kpi.prbUtilization'), key: 'kpi-ul-prb' },
      ],
    },
    {
      title: t('perf.query.handoverKpi'),
      key: 'cat-3',
      children: [
        { title: t('kpi.handoverSuccessRate'), key: 'kpi-ho-sr' },
        { title: t('kpi.handoverFailures'), key: 'kpi-ho-fail' },
      ],
    },
    {
      title: t('perf.query.userKpi'),
      key: 'cat-4',
      children: [
        { title: t('kpi.maxUsers'), key: 'kpi-max-user' },
        { title: t('kpi.onlineUsers'), key: 'kpi-active-user' },
        { title: t('kpi.idleUsers'), key: 'kpi-idle-user' },
      ],
    },
  ], [t]);

  // 表单状态
  const [reportPeriod, setReportPeriod] = useState<'15' | '60' | '1440'>('15');
  const [checkAll, setCheckAll] = useState<'0' | '1'>('1');
  const [week, setWeek] = useState<string[]>([]);
  const [hour, setHour] = useState<string[]>([]);
  const [busyTime, setBusyTime] = useState<'' | '6' | '8'>('');
  const [selDeviceType, setSelDeviceType] = useState<'1' | '2'>('2');
  const [indicatorLevel, setIndicatorLevel] = useState<'device' | 'plmn'>('device');
  const [selectedKpis, setSelectedKpis] = useState<string[]>([]);
  const [isDefault, setIsDefault] = useState(false);

  // 搜索状态
  const [deviceSearchText, setDeviceSearchText] = useState('');
  const [kpiSearchText, setKpiSearchText] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string[]>([]);

  // 是否只读模式
  const isReadOnly = mode === 'view';

  // 关闭抽屉
  const handleClose = () => {
    onClose();
  };

  // 提交表单
  const handleSubmit = () => {
    if (isReadOnly) {
      handleClose();
      return;
    }

    form.validateFields().then((values) => {
      const submitData: TemplateFormValues = {
        ...values,
        reportPeriod,
        checkAll,
        week: checkAll === '1' ? [] : week,
        hour: checkAll === '1' ? [] : hour,
        busyTime,
        selDeviceType,
        indicatorLevel,
        selectedKpis,
        isDefault,
      };

      // 验证必填项
      if (selectedKpis.length === 0) {
        message.warning(t('perf.query.pleaseSelectKpi'));
        return;
      }

      onSubmit(submitData);
    });
  };

  // 查询粒度变化
  const handleReportPeriodChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value as '15' | '60' | '1440';
    setReportPeriod(value);
    if (value === '1440') {
      setCheckAll('1');
      setWeek([]);
      setHour([]);
      setBusyTime('');
    }
  };

  // 时段模式变化
  const handleCheckAllChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.checked ? '1' : '0';
    setCheckAll(value);
    if (value === '1') {
      setWeek([]);
      setHour([]);
      setBusyTime('');
    }
  };

  // 忙时选择变化
  const handleBusyTimeChange = (value: '' | '6' | '8') => {
    setBusyTime(value);
    if (value) {
      setHour(BUSY_HOUR_PRESETS[value]);
      setWeek(['1', '2', '3', '4', '5', '6', '7']);
    }
  };

  // 周选择变化
  const handleWeekChange = (checkedValues: string[]) => {
    setWeek(checkedValues);
    setBusyTime(''); // 手动选择时清除忙时预设
  };

  // 小时选择变化
  const handleHourChange = (checkedValues: string[]) => {
    setHour(checkedValues);
    setBusyTime(''); // 手动选择时清除忙时预设
  };

  // KPI 树选择
  const handleKpiTreeCheck: TreeProps['onCheck'] = (checkedKeys) => {
    const keys = Array.isArray(checkedKeys) ? checkedKeys : checkedKeys.checked;
    setSelectedKpis(keys as string[]);
  };

  // 获取抽屉标题
  const getDrawerTitle = () => {
    switch (mode) {
      case 'add':
        return t('perf.query.addTemplate');
      case 'edit':
        return t('perf.query.editTemplate');
      case 'view':
        return t('perf.query.viewTemplate');
      case 'copy':
        return t('perf.query.copyTemplate');
      default:
        return t('perf.query.addTemplate');
    }
  };

  // 是否显示时段配置
  const showTimeConfig = reportPeriod !== '1440';

  // 时段配置是否禁用
  const timeConfigDisabled = checkAll === '1' || reportPeriod === '1440';

  // 过滤后的 KPI 列表
  const filteredKpiCategories = useMemo(() => {
    if (!kpiSearchText) return MOCK_KPI_CATEGORIES;

    const filterTree = (nodes: DataNode[]): DataNode[] => {
      return nodes.reduce((acc: DataNode[], node) => {
        const title = String(node.title);
        if (title.toLowerCase().includes(kpiSearchText.toLowerCase())) {
          acc.push(node);
        } else if (node.children) {
          const filteredChildren = filterTree(node.children);
          if (filteredChildren.length > 0) {
            acc.push({ ...node, children: filteredChildren });
          }
        }
        return acc;
      }, []);
    };

    return filterTree(MOCK_KPI_CATEGORIES);
  }, [kpiSearchText, MOCK_KPI_CATEGORIES]);

  return (
    <Drawer
      title={getDrawerTitle()}
      open={open}
      onClose={handleClose}
      width={720}
      destroyOnClose
      styles={{
        body: { padding: 0 },
      }}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          {!isReadOnly && (
            <Checkbox
              checked={isDefault}
              onChange={(e) => setIsDefault(e.target.checked)}
            >
              {t('perf.query.setAsDefault')}
            </Checkbox>
          )}
          <Space>
            <Button onClick={handleClose}>{t('common.cancel')}</Button>
            {!isReadOnly && (
              <Button type="primary" loading={loading} onClick={handleSubmit}>
                {t('common.confirm')}
              </Button>
            )}
          </Space>
        </div>
      }
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={DEFAULT_FORM_VALUES}
        disabled={isReadOnly}
      >
        {/* 基本信息 */}
        <Card
          title={t('perf.query.basicInfo')}
          size="small"
          style={{ marginBottom: 16, borderRadius: 0 }}
          styles={{ body: { padding: '16px 24px' } }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="tempName"
                label={t('perf.query.templateName')}
                rules={[{ required: true, message: t('common.pleaseInput') }]}
              >
                <Input
                  placeholder={t('perf.query.templateNamePlaceholder')}
                  maxLength={50}
                  showCount
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label=" ">
                <Space>
                  <Checkbox
                    checked={form.getFieldValue('isPublic') === '1'}
                    onChange={(e) => form.setFieldValue('isPublic', e.target.checked ? '1' : '0')}
                  >
                    {t('perf.query.setAsPublic')}
                  </Checkbox>
                  <Tooltip title={t('perf.query.publicTemplateTip')}>
                    <InfoCircleOutlined style={{ color: token.colorTextSecondary }} />
                  </Tooltip>
                </Space>
              </Form.Item>
            </Col>
          </Row>

          {/* 详情模式下显示创建人/更新人 */}
          {isReadOnly && (
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item label={t('perf.query.creator')}>
                  <Input value={initialValues?.creator || '-'} disabled />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label={t('perf.query.updator')}>
                  <Input value={initialValues?.updator || '-'} disabled />
                </Form.Item>
              </Col>
            </Row>
          )}

          <Form.Item name="description" label={t('common.description')}>
            <TextArea
              placeholder={t('common.pleaseInput')}
              maxLength={500}
              rows={3}
              showCount
            />
          </Form.Item>
        </Card>

        {/* 周期设定 */}
        <Card
          title={t('perf.query.periodSetting')}
          size="small"
          style={{ marginBottom: 16, borderRadius: 0 }}
          styles={{ body: { padding: '16px 24px' } }}
        >
          <Form.Item label={t('perf.query.granularity')} required>
            <Radio.Group value={reportPeriod} onChange={handleReportPeriodChange}>
              <Radio.Button value="15">15Min</Radio.Button>
              <Radio.Button value="60">60Min</Radio.Button>
              <Radio.Button value="1440">24hour</Radio.Button>
            </Radio.Group>
          </Form.Item>

          {showTimeConfig && (
            <>
              <Form.Item>
                <Space>
                  <Checkbox
                    checked={checkAll === '1'}
                    onChange={handleCheckAllChange}
                  >
                    {t('perf.query.allTimeSlots')}
                  </Checkbox>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('perf.query.allTimeSlotsTip')}
                  </Text>
                </Space>
              </Form.Item>

              <div
                style={{
                  padding: 16,
                  background: token.colorBgLayout,
                  borderRadius: 8,
                  opacity: timeConfigDisabled ? 0.5 : 1,
                }}
              >
                {/* 周选择 */}
                <div style={{ marginBottom: 16 }}>
                  <Text strong style={{ display: 'block', marginBottom: 8 }}>
                    {t('perf.query.week')}
                  </Text>
                  <Checkbox.Group
                    value={week}
                    onChange={handleWeekChange}
                    disabled={timeConfigDisabled}
                  >
                    <Row>
                      {WEEK_OPTIONS.map((item) => (
                        <Col key={item.value} span={3}>
                          <Checkbox value={item.value}>{item.label}</Checkbox>
                        </Col>
                      ))}
                    </Row>
                  </Checkbox.Group>
                </div>

                {/* 忙时快捷选择 */}
                <div style={{ marginBottom: 16 }}>
                  <Text strong style={{ display: 'block', marginBottom: 8 }}>
                    {t('perf.query.busyHour')}
                  </Text>
                  <Space>
                    <Select
                      value={busyTime}
                      onChange={handleBusyTimeChange}
                      placeholder={t('perf.query.selectBusyHour')}
                      style={{ width: 120 }}
                      disabled={timeConfigDisabled}
                      allowClear
                    >
                      <Select.Option value="6">{t('perf.query.busyHour6')}</Select.Option>
                      <Select.Option value="8">{t('perf.query.busyHour8')}</Select.Option>
                    </Select>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {t('perf.query.busyHourTip')}
                    </Text>
                  </Space>
                </div>

                {/* 小时选择 */}
                <div>
                  <Text strong style={{ display: 'block', marginBottom: 8 }}>
                    {t('perf.query.hour')}
                  </Text>
                  <Checkbox.Group
                    value={hour}
                    onChange={handleHourChange}
                    disabled={timeConfigDisabled}
                  >
                    <Row>
                      {HOUR_OPTIONS.map((item) => (
                        <Col key={item.value} span={3}>
                          <Checkbox value={item.value}>{item.label}</Checkbox>
                        </Col>
                      ))}
                    </Row>
                  </Checkbox.Group>
                </div>
              </div>
            </>
          )}
        </Card>

        {/* 设备列表 */}
        <Card
          title={t('perf.query.deviceList')}
          size="small"
          style={{ marginBottom: 16, borderRadius: 0 }}
          styles={{ body: { padding: '16px 24px' } }}
        >
          <Form.Item label={t('perf.query.deviceSelectType')} required>
            <Radio.Group
              value={selDeviceType}
              onChange={(e) => setSelDeviceType(e.target.value)}
            >
              <Radio value="1">{t('device.group')}</Radio>
              <Radio value="2">{t('device.name')}</Radio>
            </Radio.Group>
            <Text type="secondary" style={{ marginLeft: 16, fontSize: 12 }}>
              {t('perf.query.deviceLimit', { count: 100 })}
            </Text>
          </Form.Item>

          {/* 设备选择区域 */}
          <div
            style={{
              border: `1px solid ${token.colorBorder}`,
              borderRadius: 8,
              padding: 12,
            }}
          >
            <Input
              placeholder={t('common.search')}
              prefix={<SearchOutlined />}
              value={deviceSearchText}
              onChange={(e) => setDeviceSearchText(e.target.value)}
              style={{ marginBottom: 12 }}
            />

            {selDeviceType === '1' ? (
              // 设备组选择
              <div style={{ height: 200, overflow: 'auto' }}>
                <Tree
                  checkable
                  checkedKeys={selectedKpis.filter(k => k.startsWith('group-'))}
                  treeData={MOCK_DEVICE_GROUPS}
                  style={{ background: 'transparent' }}
                />
              </div>
            ) : (
              // 设备选择
              <div style={{ height: 200, overflow: 'auto' }}>
                <Tree
                  checkable
                  checkedKeys={selectedKpis.filter(k => k.startsWith('ENB') || k.startsWith('GNB'))}
                  treeData={MOCK_DEVICE_GROUPS.map(g => ({
                    ...g,
                    children: MOCK_DEVICES.filter(d => d.key.startsWith(g.key.split('-')[0]))
                      .map(d => ({ key: d.key, title: d.title })),
                  }))}
                  style={{ background: 'transparent' }}
                />
              </div>
            )}
          </div>
        </Card>

        {/* 指标列表 */}
        <Card
          title={t('perf.query.kpiList')}
          size="small"
          style={{ marginBottom: 16, borderRadius: 0 }}
          styles={{ body: { padding: '16px 24px' } }}
        >
          {/* 等级选择（仅 eNB） */}
          {networkType === 'enb' && (
            <Form.Item label={t('perf.query.indicatorLevel')}>
              <Radio.Group
                value={indicatorLevel}
                onChange={(e) => setIndicatorLevel(e.target.value)}
              >
                <Radio value="device">{t('perf.query.indicatorLevelDevice')}</Radio>
                <Radio value="plmn">{t('perf.query.indicatorLevelPlmn')}</Radio>
              </Radio.Group>
            </Form.Item>
          )}

          {/* KPI 选择区域 */}
          <div
            style={{
              border: `1px solid ${token.colorBorder}`,
              borderRadius: 8,
              padding: 12,
            }}
          >
            <Space style={{ marginBottom: 12, width: '100%' }} direction="vertical">
              {/* 功能集筛选 */}
              <Select
                placeholder={t('perf.query.selectCategory')}
                value={selectedCategory}
                onChange={setSelectedCategory}
                style={{ width: '100%' }}
                allowClear
                options={[
                  { label: t('perf.query.accessKpi'), value: 'cat-1' },
                  { label: t('perf.query.throughputKpi'), value: 'cat-2' },
                  { label: t('perf.query.handoverKpi'), value: 'cat-3' },
                  { label: t('perf.query.userKpi'), value: 'cat-4' },
                ]}
              />
              {/* 搜索 */}
              <Input
                placeholder={t('perf.query.kpiSearchPlaceholder')}
                prefix={<SearchOutlined />}
                value={kpiSearchText}
                onChange={(e) => setKpiSearchText(e.target.value)}
              />
            </Space>

            {/* KPI 树 */}
            <div style={{ height: 250, overflow: 'auto' }}>
              {filteredKpiCategories.length > 0 ? (
                <Tree
                  checkable
                  checkedKeys={selectedKpis}
                  onCheck={handleKpiTreeCheck}
                  treeData={filteredKpiCategories}
                  style={{ background: 'transparent' }}
                  defaultExpandedKeys={['cat-1', 'cat-2']}
                />
              ) : (
                <Empty description={t('common.noData')} />
              )}
            </div>

            {/* 已选数量提示 */}
            <div
              style={{
                marginTop: 12,
                padding: '8px 12px',
                background: token.colorPrimaryBg,
                borderRadius: 4,
              }}
            >
              <Text>
                {t('perf.query.selectedKpiCount', { count: selectedKpis.length })}
              </Text>
            </div>
          </div>
        </Card>
      </Form>
    </Drawer>
  );
}
