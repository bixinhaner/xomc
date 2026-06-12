import { useMemo } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { Button, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ArrowLeftOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';

interface UeRecord {
  key: string;
  ueId: number;
  imsi: string;
  vmac: string;
  cpeName: string;
  downlinkRate: number;
  uplinkRate: number;
  ip: string;
  port: number;
  ulsinr: number;
  dlcqi?: number;
  ulmcs: number;
  dlmcs?: number;
  txpower: number;
  uplinkBler: number;
  downlinkBler?: number;
  pathloss: number;
  ueS1apId?: number;
  mmeS1apId?: number;
}

/** 生成 Mock UE 数据 */
function generateMockUeData(count: number): UeRecord[] {
  return Array.from({ length: count }, (_, i) => ({
    key: String(i),
    ueId: 1000 + i,
    imsi: `46000${String(Math.floor(Math.random() * 1e10)).padStart(10, '0')}`,
    vmac: Array.from({ length: 6 }, () => Math.floor(Math.random() * 256).toString(16).padStart(2, '0')).join(':'),
    cpeName: `CPE-${String(i + 1).padStart(3, '0')}`,
    downlinkRate: +(Math.random() * 150).toFixed(1),
    uplinkRate: +(Math.random() * 50).toFixed(1),
    ip: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 254) + 1}`,
    port: 1024 + Math.floor(Math.random() * 64000),
    ulsinr: +(Math.random() * 30 - 5).toFixed(1),
    ulmcs: Math.floor(Math.random() * 28),
    txpower: +(Math.random() * 23).toFixed(1),
    uplinkBler: +(Math.random() * 10).toFixed(2),
    pathloss: +(80 + Math.random() * 40).toFixed(1),
    ueS1apId: Math.floor(Math.random() * 65535),
    mmeS1apId: Math.floor(Math.random() * 65535),
    dlcqi: Math.floor(Math.random() * 15),
    dlmcs: Math.floor(Math.random() * 28),
    downlinkBler: +(Math.random() * 10).toFixed(2),
  }));
}

export default function UeDetail() {
  const navigate = useNavigate();
  const { sn } = useParams<{ sn: string }>();
  const [searchParams] = useSearchParams();
  const deviceName = searchParams.get('name') ?? sn ?? '';
  const ueCount = Number(searchParams.get('ueCount') ?? 0);

  const columns = useMemo<ColumnsType<UeRecord>>(() => [
    { title: 'UEID', dataIndex: 'ueId', width: 80, fixed: 'left' },
    { title: 'IMSI', dataIndex: 'imsi', width: 140 },
    { title: 'VMAC', dataIndex: 'vmac', width: 130 },
    { title: 'CPE名称', dataIndex: 'cpeName', width: 110 },
    { title: '下行速率(Mbps)', dataIndex: 'downlinkRate', width: 120, align: 'right' },
    { title: '上行速率(Mbps)', dataIndex: 'uplinkRate', width: 120, align: 'right' },
    { title: 'IP地址', dataIndex: 'ip', width: 130 },
    { title: '端口', dataIndex: 'port', width: 70, align: 'right' },
    { title: '上行SINR(dB)', dataIndex: 'ulsinr', width: 110, align: 'right' },
    { title: '下行CQI', dataIndex: 'dlcqi', width: 80, align: 'right' },
    { title: '上行MCS', dataIndex: 'ulmcs', width: 80, align: 'right' },
    { title: '下行MCS', dataIndex: 'dlmcs', width: 80, align: 'right' },
    { title: '发送功率(dBm)', dataIndex: 'txpower', width: 110, align: 'right' },
    { title: '上行BLER(%)', dataIndex: 'uplinkBler', width: 100, align: 'right' },
    { title: '下行BLER(%)', dataIndex: 'downlinkBler', width: 110, align: 'right' },
    { title: '路径损耗(dBm)', dataIndex: 'pathloss', width: 120, align: 'right' },
    { title: 'UE_S1AP_ID', dataIndex: 'ueS1apId', width: 110, align: 'right' },
    { title: 'MME_S1AP_ID', dataIndex: 'mmeS1apId', width: 120, align: 'right' },
  ], []);

  const mockData = useMemo(
    () => generateMockUeData(Math.min(ueCount, 50)),
    [ueCount],
  );

  return (
    <ListPageLayout
      title={`UE 详情 — ${deviceName}`}
      extra={
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)}>
          返回
        </Button>
      }
    >
      <Table<UeRecord>
        columns={columns}
        dataSource={mockData}
        size="small"
        scroll={{ x: 1600 }}
        pagination={mockData.length > 20 ? { pageSize: 20, showSizeChanger: false } : false}
      />
    </ListPageLayout>
  );
}
