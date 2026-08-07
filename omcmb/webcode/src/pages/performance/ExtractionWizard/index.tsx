import { useState, useMemo } from 'react';
import { Button, DatePicker, Form, Select, Space, Table, Typography, message } from 'antd';
import WizardPageLayout from '@/components/Layout/WizardPageLayout';
import TransferBox from '@/components/TransferBox';
import type { TransferItem } from '@/components/TransferBox';
import SchedulePicker from '@/components/SchedulePicker';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import { useAllKPIs, useMultipleKPISeries } from '@core/hooks/api/usePerformance';
import { useT } from '@/hooks/useT';

const { RangePicker } = DatePicker;

const GRANULARITY_OPTIONS = [
  { label: '15min', value: '15min' },
  { label: '30min', value: '30min' },
  { label: '1h', value: '1h' },
  { label: '1d', value: '1d' },
];

interface ResultRow extends Record<string, unknown> {
  id: string;
  stationName: string;
  sn: string;
  kpiCode: string;
  kpiName: string;
  value: number;
  unit: string;
  timestamp: string;
}

const MOCK_RESULTS_RAW = [
  { id: '1', stationName: '北京朝阳基站01', sn: 'ENB00001', kpiCode: 'RRC_SR', kpiNameKey: 'kpi.rrcSetupSuccessRate', value: 99.2, unit: '%', timestamp: '2026-03-02 10:00:00' },
  { id: '2', stationName: '北京朝阳基站01', sn: 'ENB00001', kpiCode: 'DL_THROUGHPUT', kpiNameKey: 'kpi.dlThroughput', value: 145.6, unit: 'Mbps', timestamp: '2026-03-02 10:00:00' },
  { id: '3', stationName: '北京海淀基站01', sn: 'ENB00002', kpiCode: 'RRC_SR', kpiNameKey: 'kpi.rrcSetupSuccessRate', value: 98.9, unit: '%', timestamp: '2026-03-02 10:00:00' },
  { id: '4', stationName: '北京海淀基站01', sn: 'ENB00002', kpiCode: 'DL_THROUGHPUT', kpiNameKey: 'kpi.dlThroughput', value: 132.1, unit: 'Mbps', timestamp: '2026-03-02 10:00:00' },
];

const DEVICE_OPTIONS = [
  { sn: 'ENB00001', name: '北京朝阳基站01' },
  { sn: 'ENB00002', name: '北京海淀基站01' },
  { sn: 'ENB00003', name: '上海浦东基站01' },
  { sn: 'GNB00001', name: '北京5G基站01' },
  { sn: 'GNB00002', name: '北京5G基站02' },
];

export default function ExtractionWizard() {
  const t = useT();
  const [currentStep, setCurrentStep] = useState(0);
  const [selectedKPIs, setSelectedKPIs] = useState<string[]>([]);
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);
  const [timeConfig, setTimeConfig] = useState<{ timeRange?: [string, string]; granularity?: string; schedule?: unknown }>({});
  const [resultView, setResultView] = useState<'table' | 'chart'>('table');
  const [scheduleValue, setScheduleValue] = useState<unknown>(null);
  const [timeForm] = Form.useForm();

  const { data: kpisData } = useAllKPIs();
  const { data: seriesData } = useMultipleKPISeries(selectedKPIs, selectedDevices[0]);

  const mockResults: ResultRow[] = useMemo(() =>
    MOCK_RESULTS_RAW.map((r) => ({ ...r, kpiName: t(r.kpiNameKey) })),
  [t]);

  const WIZARD_STEPS = useMemo(() => [
    { title: t('perf.kpiName'), description: t('common.pleaseSelect') },
    { title: t('device.name'), description: t('common.pleaseSelect') },
    { title: t('perf.timeRange'), description: t('perf.granularity') },
    { title: t('table.result'), description: t('common.view') },
  ], [t]);

  const kpiTransferItems: TransferItem[] = (kpisData ?? []).map((k) => ({
    key: k.kpiCode,
    label: k.kpiName,
    description: `${k.category} · ${k.unit}`,
  })).concat(
    selectedKPIs.length === 0
      ? [
          { key: 'RRC_SR', label: t('kpi.rrcSetupSuccessRate'), description: `${t('kpi.tree.radioAccess')} · %` },
          { key: 'ERAB_SR', label: t('kpi.erabSetupSuccessRate'), description: `${t('kpi.tree.radioAccess')} · %` },
          { key: 'HO_SR', label: t('kpi.handoverSuccessRate'), description: `${t('kpi.tree.handover')} · %` },
          { key: 'DL_THROUGHPUT', label: t('kpi.dlThroughput'), description: `${t('kpi.dlThroughput')} · Mbps` },
          { key: 'UL_THROUGHPUT', label: t('kpi.ulThroughput'), description: `${t('kpi.ulThroughput')} · Mbps` },
          { key: 'MAX_USERS', label: t('kpi.maxUsers'), description: `${t('kpi.tree.userCount')} · 个` },
          { key: 'AVAILABILITY', label: t('kpi.availability'), description: `${t('kpi.tree.quality')} · %` },
        ]
      : []
  );

  const resultColumns: DataTableColumn<ResultRow>[] = useMemo(() => [
    { key: 'stationName', title: t('device.name'), dataIndex: 'stationName', width: 180 },
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 120, mono: true },
    { key: 'kpiCode', title: t('perf.kpiCode'), dataIndex: 'kpiCode', width: 140, mono: true },
    { key: 'kpiName', title: t('perf.kpiName'), dataIndex: 'kpiName', width: 160 },
    { key: 'value', title: t('perf.value'), dataIndex: 'value', width: 100 },
    { key: 'unit', title: t('perf.unit'), dataIndex: 'unit', width: 70 },
    { key: 'timestamp', title: t('perf.timestamp'), dataIndex: 'timestamp', width: 160 },
  ], [t]);

  const chartSeries = (seriesData ?? []).map((s) => ({
    name: s.kpiName,
    data: s.data.map((d) => d.value),
    xData: s.data.map((d) => d.timestamp),
  }));

  const goNext = () => {
    if (currentStep === 0 && selectedKPIs.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    if (currentStep === 1 && selectedDevices.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    if (currentStep === 2) {
      timeForm.validateFields().then((vals: Record<string, unknown>) => {
        const range = vals.timeRange as [{ format: (f: string) => string }, { format: (f: string) => string }] | undefined;
        setTimeConfig({
          timeRange: range ? [range[0].format('YYYY-MM-DD HH:mm:ss'), range[1].format('YYYY-MM-DD HH:mm:ss')] : undefined,
          granularity: vals.granularity as string,
          schedule: scheduleValue,
        });
        setCurrentStep((s) => s + 1);
      }).catch(() => undefined);
      return;
    }
    setCurrentStep((s) => s + 1);
  };

  const goPrev = () => setCurrentStep((s) => s - 1);

  const renderStep = () => {
    switch (currentStep) {
      case 0:
        return (
          <div>
            <Typography.Title level={5} style={{ marginBottom: 16 }}>{t('perf.kpiName')}</Typography.Title>
            <TransferBox
              dataSource={kpiTransferItems}
              targetKeys={selectedKPIs}
              onChange={setSelectedKPIs}
              titles={[t('common.all'), t('common.pleaseSelect')]}
              searchable
            />
          </div>
        );
      case 1:
        return (
          <div>
            <Typography.Title level={5} style={{ marginBottom: 16 }}>{t('device.name')}</Typography.Title>
            <Table
              size="small"
              dataSource={DEVICE_OPTIONS}
              rowKey="sn"
              rowSelection={{
                selectedRowKeys: selectedDevices,
                onChange: (keys) => setSelectedDevices(keys as string[]),
              }}
              columns={[
                { title: t('device.sn'), dataIndex: 'sn', key: 'sn', width: 140 },
                { title: t('device.name'), dataIndex: 'name', key: 'name' },
              ]}
              pagination={false}
              style={{ maxWidth: 600 }}
            />
          </div>
        );
      case 2:
        return (
          <div>
            <Typography.Title level={5} style={{ marginBottom: 16 }}>{t('perf.timeRange')}</Typography.Title>
            <Form form={timeForm} layout="vertical" style={{ maxWidth: 500 }}>
              <Form.Item label={t('perf.timeRange')} name="timeRange" rules={[{ required: true, message: t('common.pleaseSelect') }]}>
                <RangePicker showTime style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item label={t('perf.granularity')} name="granularity" rules={[{ required: true, message: t('common.pleaseSelect') }]} initialValue="1h">
                <Select options={GRANULARITY_OPTIONS} />
              </Form.Item>
              <Form.Item label={t('common.more')}>
                <SchedulePicker value={scheduleValue as Parameters<typeof SchedulePicker>[0]['value']} onChange={setScheduleValue} />
              </Form.Item>
            </Form>
            {timeConfig.granularity && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {selectedKPIs.length} KPI, {selectedDevices.length} {t('device.name')}, {t('perf.granularity')}: {timeConfig.granularity}
              </Typography.Text>
            )}
          </div>
        );
      case 3:
        return (
          <div>
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography.Title level={5} style={{ margin: 0 }}>{t('table.result')}</Typography.Title>
              <Space>
                <Button.Group>
                  <Button type={resultView === 'table' ? 'primary' : 'default'} onClick={() => setResultView('table')}>{t('common.view')}</Button>
                  <Button type={resultView === 'chart' ? 'primary' : 'default'} onClick={() => setResultView('chart')}>{t('nav.performance.charts')}</Button>
                </Button.Group>
                <Button onClick={() => void message.info(t('common.exportInProgress'))}>{t('common.export')}</Button>
              </Space>
            </div>
            {resultView === 'table' ? (
              <DataTable<ResultRow>
                tableId="extraction-result"
                columns={resultColumns}
                dataSource={mockResults}
                rowKey="id"
                showPagination={false}
                scroll={{ x: 900 }}
              />
            ) : (
              <div style={{ height: 400 }}>
                {chartSeries.length > 0 ? (
                  <LineChart
                    series={chartSeries.map((s) => ({ name: s.name, data: s.data }))}
                    xData={chartSeries[0]?.xData ?? []}
                    height={380}
                    connectNulls
                  />
                ) : (
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: '#8c8c8c' }}>
                    {t('common.noData')}
                  </div>
                )}
              </div>
            )}
          </div>
        );
      default:
        return null;
    }
  };

  const footer = (
    <Space>
      {currentStep > 0 && <Button onClick={goPrev}>{t('common.prev')}</Button>}
      {currentStep < WIZARD_STEPS.length - 1 && (
        <Button type="primary" onClick={goNext}>
          {t('common.next')}
        </Button>
      )}
      {currentStep === WIZARD_STEPS.length - 1 && (
        <Button type="primary" onClick={() => { setCurrentStep(0); setSelectedKPIs([]); setSelectedDevices([]); }}>
          {t('common.refresh')}
        </Button>
      )}
    </Space>
  );

  return (
    <WizardPageLayout steps={WIZARD_STEPS} currentStep={currentStep} footer={footer}>
      {renderStep()}
    </WizardPageLayout>
  );
}
