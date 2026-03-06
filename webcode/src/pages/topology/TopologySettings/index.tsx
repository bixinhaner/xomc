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
      label: '布局算法',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="布局算法" name="layoutAlgorithm">
            <Select
              options={[
                { label: '力导向布局 (Force-Directed)', value: 'force' },
                { label: '树形布局 (Tree)', value: 'tree' },
                { label: '环形布局 (Circular)', value: 'circular' },
                { label: '分层布局 (Hierarchy)', value: 'hierarchy' },
              ]}
              onChange={(val) => setLayoutAlgo(val as string)}
            />
          </Form.Item>
          {layoutAlgo === 'force' && (
            <>
              <Form.Item label="斥力强度" name="forceStrength">
                <InputNumber style={{ width: '100%' }} min={-1000} max={0} />
              </Form.Item>
              <Form.Item label="链接距离" name="forceDistance">
                <Slider min={50} max={400} />
              </Form.Item>
              <Form.Item label="引力强度" name="forceGravity">
                <InputNumber style={{ width: '100%' }} min={0} max={1} step={0.05} />
              </Form.Item>
            </>
          )}
        </div>
      ),
    },
    {
      key: 'node',
      label: '节点样式',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="节点大小 (px)" name="nodeSize">
            <Slider min={12} max={48} marks={{ 12: '12', 24: '24', 36: '36', 48: '48' }} />
          </Form.Item>
          <Form.Item label="节点形状" name="nodeShape">
            <Select options={[{ label: '圆形', value: 'circle' }, { label: '矩形', value: 'rect' }, { label: '菱形', value: 'diamond' }]} />
          </Form.Item>
          <Form.Item label="节点不透明度" name="nodeOpacity">
            <Slider min={0.3} max={1} step={0.1} marks={{ 0.3: '30%', 0.7: '70%', 1: '100%' }} />
          </Form.Item>
          <Form.Item label="节点字体大小" name="nodeFontSize">
            <InputNumber min={8} max={20} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="显示节点标签" name="showNodeLabel" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'edge',
      label: '连线样式',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="连线宽度 (px)" name="edgeWidth">
            <Slider min={1} max={6} marks={{ 1: '1', 3: '3', 6: '6' }} />
          </Form.Item>
          <Form.Item label="连线样式" name="edgeStyle">
            <Select options={[{ label: '实线', value: 'solid' }, { label: '虚线', value: 'dashed' }, { label: '点线', value: 'dotted' }]} />
          </Form.Item>
          <Form.Item label="连线弯曲度" name="edgeCurvature">
            <Slider min={0} max={0.8} step={0.1} marks={{ 0: '直线', 0.4: '弯曲', 0.8: '强弯' }} />
          </Form.Item>
          <Form.Item label="显示连线标签" name="showEdgeLabel" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="显示箭头" name="edgeArrow" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'display',
      label: '显示选项',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="显示状态标记" name="showStatusBadge" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="显示设备类型" name="showDeviceType" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="显示告警数量" name="showAlarmCount" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="启用动画效果" name="enableAnimation" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="动画时长 (ms)" name="animationDuration">
            <InputNumber min={100} max={2000} step={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="自动刷新间隔 (秒)" name="autoRefreshInterval">
            <Select
              options={[
                { label: '不自动刷新', value: 0 },
                { label: '10秒', value: 10 },
                { label: '30秒', value: 30 },
                { label: '60秒', value: 60 },
                { label: '5分钟', value: 300 },
              ]}
            />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'interaction',
      label: '交互设置',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="允许拖拽节点" name="enableDrag" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="允许缩放" name="enableZoom" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="允许平移" name="enablePan" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="显示悬浮提示" name="enableTooltip" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
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
          以下设置将影响拓扑图的显示效果，修改后需保存才能生效。
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
