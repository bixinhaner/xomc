import { useState } from 'react';
import { Button, Card, Collapse, Form, InputNumber, Select, Slider, Space, Switch, Typography, message } from 'antd';
import { SaveOutlined, ReloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

interface TopologySettingsValues {
  // 布局算法
  layoutAlgorithm: 'force' | 'tree' | 'circular' | 'hierarchy';
  forceStrength: number;
  forceDistance: number;
  forceGravity: number;
  // 节点样式
  nodeSize: number;
  nodeShape: 'circle' | 'rect' | 'diamond';
  nodeOpacity: number;
  showNodeLabel: boolean;
  nodeFontSize: number;
  // 边样式
  edgeWidth: number;
  edgeStyle: 'solid' | 'dashed' | 'dotted';
  edgeCurvature: number;
  showEdgeLabel: boolean;
  edgeArrow: boolean;
  // 显示选项
  showStatusBadge: boolean;
  showDeviceType: boolean;
  showAlarmCount: boolean;
  enableAnimation: boolean;
  animationDuration: number;
  // 交互
  enableDrag: boolean;
  enableZoom: boolean;
  enablePan: boolean;
  enableTooltip: boolean;
  autoRefreshInterval: number;
}

const DEFAULT_VALUES: TopologySettingsValues = {
  layoutAlgorithm: 'force',
  forceStrength: -300,
  forceDistance: 150,
  forceGravity: 0.1,
  nodeSize: 24,
  nodeShape: 'circle',
  nodeOpacity: 1,
  showNodeLabel: true,
  nodeFontSize: 12,
  edgeWidth: 2,
  edgeStyle: 'solid',
  edgeCurvature: 0,
  showEdgeLabel: false,
  edgeArrow: true,
  showStatusBadge: true,
  showDeviceType: true,
  showAlarmCount: true,
  enableAnimation: true,
  animationDuration: 500,
  enableDrag: true,
  enableZoom: true,
  enablePan: true,
  enableTooltip: true,
  autoRefreshInterval: 30,
};

export default function TopologySettings() {
  const t = useT();
  const [form] = Form.useForm<TopologySettingsValues>();
  const [saving, setSaving] = useState(false);
  const [layoutAlgo, setLayoutAlgo] = useState<string>('force');

  const handleSave = () => {
    form.validateFields().then((vals) => {
      setSaving(true);
      setTimeout(() => {
        setSaving(false);
        void message.success(t('common.save'));
        console.log('topology settings:', vals);
      }, 600);
    }).catch(() => undefined);
  };

  const handleReset = () => {
    form.setFieldsValue(DEFAULT_VALUES);
    setLayoutAlgo('force');
    void message.info(t('common.reset'));
  };

  const collapseItems = [
    {
      key: 'layout',
      label: t('topology.settings.layoutAlgorithm'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('topology.settings.layoutAlgorithm')} name="layoutAlgorithm">
            <Select
              options={[
                { label: t('topology.settings.forceDirected'), value: 'force' },
                { label: t('topology.settings.treeLayout'), value: 'tree' },
                { label: t('topology.settings.circularLayout'), value: 'circular' },
                { label: t('topology.settings.hierarchyLayout'), value: 'hierarchy' },
              ]}
              onChange={(val) => setLayoutAlgo(val as string)}
            />
          </Form.Item>
          {layoutAlgo === 'force' && (
            <>
              <Form.Item label={t('topology.settings.forceStrength')} name="forceStrength">
                <InputNumber style={{ width: '100%' }} min={-1000} max={0} />
              </Form.Item>
              <Form.Item label={t('topology.settings.linkDistance')} name="forceDistance">
                <Slider min={50} max={400} />
              </Form.Item>
              <Form.Item label={t('topology.settings.gravityStrength')} name="forceGravity">
                <InputNumber style={{ width: '100%' }} min={0} max={1} step={0.05} />
              </Form.Item>
            </>
          )}
        </div>
      ),
    },
    {
      key: 'node',
      label: t('topology.settings.nodeStyle'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('topology.settings.nodeSize')} name="nodeSize">
            <Slider min={12} max={48} marks={{ 12: '12', 24: '24', 36: '36', 48: '48' }} />
          </Form.Item>
          <Form.Item label={t('topology.settings.nodeShape')} name="nodeShape">
            <Select options={[{ label: t('topology.settings.circle'), value: 'circle' }, { label: t('topology.settings.rect'), value: 'rect' }, { label: t('topology.settings.diamond'), value: 'diamond' }]} />
          </Form.Item>
          <Form.Item label={t('topology.settings.nodeOpacity')} name="nodeOpacity">
            <Slider min={0.3} max={1} step={0.1} marks={{ 0.3: '30%', 0.7: '70%', 1: '100%' }} />
          </Form.Item>
          <Form.Item label={t('topology.settings.nodeFontSize')} name="nodeFontSize">
            <InputNumber min={8} max={20} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('topology.settings.showNodeLabel')} name="showNodeLabel" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'edge',
      label: t('topology.settings.edgeStyle'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('topology.settings.edgeWidth')} name="edgeWidth">
            <Slider min={1} max={6} marks={{ 1: '1', 3: '3', 6: '6' }} />
          </Form.Item>
          <Form.Item label={t('topology.settings.edgeLineStyle')} name="edgeStyle">
            <Select options={[{ label: t('topology.settings.solid'), value: 'solid' }, { label: t('topology.settings.dashed'), value: 'dashed' }, { label: t('topology.settings.dotted'), value: 'dotted' }]} />
          </Form.Item>
          <Form.Item label={t('topology.settings.edgeCurvature')} name="edgeCurvature">
            <Slider min={0} max={0.8} step={0.1} marks={{ 0: t('topology.settings.straight'), 0.4: t('topology.settings.curved'), 0.8: t('topology.settings.strongCurve') }} />
          </Form.Item>
          <Form.Item label={t('topology.settings.showEdgeLabel')} name="showEdgeLabel" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('topology.settings.showArrow')} name="edgeArrow" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'display',
      label: t('topology.settings.displayOptions'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('topology.settings.showStatusBadge')} name="showStatusBadge" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('topology.settings.showDeviceType')} name="showDeviceType" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('topology.settings.showAlarmCount')} name="showAlarmCount" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('topology.settings.enableAnimation')} name="enableAnimation" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('topology.settings.animationDuration')} name="animationDuration">
            <InputNumber min={100} max={2000} step={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('topology.settings.autoRefreshInterval')} name="autoRefreshInterval">
            <Select
              options={[
                { label: t('topology.settings.noAutoRefresh'), value: 0 },
                { label: `10${t('topology.settings.seconds')}`, value: 10 },
                { label: `30${t('topology.settings.seconds')}`, value: 30 },
                { label: `60${t('topology.settings.seconds')}`, value: 60 },
                { label: '5min', value: 300 },
              ]}
            />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'interaction',
      label: t('topology.settings.interaction'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('topology.settings.enableDrag')} name="enableDrag" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('topology.settings.enableZoom')} name="enableZoom" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('topology.settings.enablePan')} name="enablePan" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('topology.settings.showTooltip')} name="enableTooltip" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
        </div>
      ),
    },
  ];

  return (
    <ListPageLayout
      title={t('nav.topology.settings')}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>{t('common.reset')}</Button>
          <Button type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving}>
            {t('common.save')}
          </Button>
        </Space>
      }
    >
      <Card>
        <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 16 }}>
          {t('topology.settings.hint')}
        </Typography.Text>
        <Form
          form={form}
          layout="vertical"
          initialValues={DEFAULT_VALUES}
          style={{ maxWidth: 960 }}
        >
          <Collapse
            defaultActiveKey={['layout', 'node', 'edge', 'display', 'interaction']}
            items={collapseItems}
            style={{ background: 'transparent' }}
          />
        </Form>
      </Card>
    </ListPageLayout>
  );
}
