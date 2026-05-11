import { Card, Empty, Typography, Space, Tag } from 'antd';
import { RadarChartOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';

const { Title, Paragraph, Text } = Typography;

export default function NetworkDiagnosis() {
  return (
    <ListPageLayout title="网络诊断" subtitle="TR-069 Diagnostics 对象统一接入（路线图 W3）">
      <Card>
        <Empty
          image={<RadarChartOutlined style={{ fontSize: 64, color: 'var(--color-primary-300)' }} />}
          imageStyle={{ height: 80 }}
          description={
            <Space direction="vertical" align="center" size={12}>
              <Title level={4} style={{ marginBottom: 0 }}>
                网络诊断子系统实施中
              </Title>
              <Paragraph type="secondary" style={{ marginBottom: 0, maxWidth: 560, textAlign: 'center' }}>
                后端 TR-181 Diagnostics 对象（IPPing / TraceRoute / Download / Upload /
                UDPEcho）接入尚未落地，参需求文档{' '}
                <Text code>docs/project/prd/F06-ops-management.md §4.4</Text>。
              </Paragraph>
              <Space size={8}>
                <Tag color="processing">路线图 W3</Tag>
                <Tag>T-0104</Tag>
                <Tag color="orange">~10 工作日</Tag>
              </Space>
            </Space>
          }
        />
      </Card>
    </ListPageLayout>
  );
}
