/**
 * SystemLicenseHistoryPage — F06 System License 重构 Step 4 历史子页。
 *
 * 路由：/license/history
 *
 * 简单分页表格：License ID / 类型 / 签名状态 / 上传时间 / 被替换时间。
 * 后端默认按 replaced_at DESC 排序。
 */
import { useMemo, useState } from 'react';
import { Button, Card, Empty, Space, Spin, Table, Tag, Typography } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

import { useT } from '@/hooks/useT';
import { useSystemLicenseHistory } from '@core/hooks/api/useSystemLicense';
import type { SystemLicenseHistory } from '@core/services/api/systemLicenseApi';
import { formatSystemTime } from '@core/utils/systemTime';

const { Title } = Typography;

function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '-';
  return formatSystemTime(iso);
}

export default function SystemLicenseHistoryPage() {
  const t = useT();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const params = useMemo(() => ({ page, pageSize }), [page, pageSize]);
  const { data, isLoading, isFetching } = useSystemLicenseHistory(params);

  const columns = [
    {
      title: t('systemLicense.history.licenseId'),
      dataIndex: 'licenseId',
      key: 'licenseId',
      width: 200,
    },
    {
      title: t('systemLicense.history.licenseType'),
      dataIndex: 'licenseType',
      key: 'licenseType',
      width: 120,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: t('systemLicense.history.signature'),
      dataIndex: 'signatureStatus',
      key: 'signatureStatus',
      width: 120,
      render: (v: SystemLicenseHistory['signatureStatus']) => {
        const color = v === 'verified' ? 'success' : v === 'invalid' ? 'error' : 'warning';
        return <Tag color={color}>{v}</Tag>;
      },
    },
    {
      title: t('systemLicense.history.uploadedAt'),
      dataIndex: 'uploadedAt',
      key: 'uploadedAt',
      width: 180,
      render: formatDateTime,
    },
    {
      title: t('systemLicense.history.replacedAt'),
      dataIndex: 'replacedAt',
      key: 'replacedAt',
      width: 180,
      render: formatDateTime,
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/license')}>
            {t('systemLicense.title')}
          </Button>
        </Space>
        <Title level={4} style={{ margin: '0 0 0 16px', flex: 1 }}>
          {t('systemLicense.history.title')}
        </Title>
      </div>

      <Card>
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin />
          </div>
        ) : !data || data.items.length === 0 ? (
          <Empty description={t('systemLicense.history.empty')} />
        ) : (
          <Table<SystemLicenseHistory>
            rowKey="id"
            dataSource={data.items}
            columns={columns}
            loading={isFetching}
            pagination={{
              current: page,
              pageSize,
              total: data.total,
              showSizeChanger: true,
              onChange: (p, ps) => {
                setPage(p);
                setPageSize(ps);
              },
            }}
          />
        )}
      </Card>
    </div>
  );
}
