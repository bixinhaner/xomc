import { Card, Empty, Typography, Space, Tag } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';

const { Title, Paragraph, Text } = Typography;

export default function Downloads() {
  return (
    <ListPageLayout title="运维下载" subtitle="设备配置 / 日志 / 诊断包按需采集（路线图 W3）">
      <Card>
        <Empty
          image={<DownloadOutlined style={{ fontSize: 64, color: 'var(--color-primary-300)' }} />}
          imageStyle={{ height: 80 }}
          description={
            <Space direction="vertical" align="center" size={12}>
              <Title level={4} style={{ marginBottom: 0 }}>
                运维下载子系统实施中
              </Title>
              <Paragraph type="secondary" style={{ marginBottom: 0, maxWidth: 580, textAlign: 'center' }}>
                设备 → OMC 的按需文件采集（配置 / 日志 / PM / MR / 诊断包 / PCAP /
                GPS 历史）尚未落地。后端依赖 transfer 模块 Upload 通道 + MinIO
                presigned URL，参需求文档{' '}
                <Text code>docs/project/prd/F06-ops-management.md §4.5</Text>。
              </Paragraph>
              <Space size={8}>
                <Tag color="processing">路线图 W3</Tag>
                <Tag>T-0105</Tag>
                <Tag color="orange">~10 工作日</Tag>
              </Space>
            </Space>
          }
        />
      </Card>
    </ListPageLayout>
  );
}
