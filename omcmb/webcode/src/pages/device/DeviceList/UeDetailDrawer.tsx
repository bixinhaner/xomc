import React, { useMemo } from 'react';
import { Drawer, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { Device } from '@/types/device';

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
  pDlcqi?: number;
  sDlcqi?: number;
  ulmcs: number;
  dlmcs?: number;
  pDlmcs?: number;
  sDlmcs?: number;
  txpower: number;
  uplinkBler: number;
  downlinkBler?: number;
  p1DownlinkBler?: number;
  p2DownlinkBler?: number;
  s1DownlinkBler?: number;
  s2DownlinkBler?: number;
  pathloss: number;
  ueS1apId?: number;
  mmeS1apId?: number;
}

interface UeDetailDrawerProps {
  device: Device | null;
  onClose: () => void;
}

/** 判断是否为 436Q 平台（双载波，使用 P/S 列变体） */
function is436Q(platformType: string | undefined): boolean {
  return !!platformType && platformType.includes('436Q');
}

/** 生成 Mock UE 数据 */
function generateMockUeData(count: number, use436Q: boolean): UeRecord[] {
  return Array.from({ length: count }, (_, i) => {
    const base: UeRecord = {
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
    };
    if (use436Q) {
      base.pDlcqi = Math.floor(Math.random() * 15);
      base.sDlcqi = Math.floor(Math.random() * 15);
      base.pDlmcs = Math.floor(Math.random() * 28);
      base.sDlmcs = Math.floor(Math.random() * 28);
      base.p1DownlinkBler = +(Math.random() * 10).toFixed(2);
      base.p2DownlinkBler = +(Math.random() * 10).toFixed(2);
      base.s1DownlinkBler = +(Math.random() * 10).toFixed(2);
      base.s2DownlinkBler = +(Math.random() * 10).toFixed(2);
    } else {
      base.dlcqi = Math.floor(Math.random() * 15);
      base.dlmcs = Math.floor(Math.random() * 28);
      base.downlinkBler = +(Math.random() * 10).toFixed(2);
    }
    return base;
  });
}

export default function UeDetailDrawer({ device, onClose }: UeDetailDrawerProps) {
  const use436Q = is436Q(device?.platformType);

  const columns = useMemo<ColumnsType<UeRecord>>(() => {
    const cols: ColumnsType<UeRecord> = [
      { title: 'UEID', dataIndex: 'ueId', width: 80, fixed: 'left' },
      { title: 'IMSI', dataIndex: 'imsi', width: 140 },
      { title: 'VMAC', dataIndex: 'vmac', width: 130 },
      { title: 'CPE名称', dataIndex: 'cpeName', width: 110 },
      {
        title: '下行速率(Mbps)',
        dataIndex: 'downlinkRate',
        width: 120,
        align: 'right',
      },
      {
        title: '上行速率(Mbps)',
        dataIndex: 'uplinkRate',
        width: 120,
        align: 'right',
      },
      { title: 'IP地址', dataIndex: 'ip', width: 130 },
      { title: '端口', dataIndex: 'port', width: 70, align: 'right' },
      {
        title: '上行SINR(dB)',
        dataIndex: 'ulsinr',
        width: 110,
        align: 'right',
      },
    ];

    // CQI 列 — 436Q 拆分为 P/S 双载波
    if (use436Q) {
      cols.push(
        { title: 'P_Dlcqi', dataIndex: 'pDlcqi', width: 80, align: 'right' },
        { title: 'S_Dlcqi', dataIndex: 'sDlcqi', width: 80, align: 'right' },
      );
    } else {
      cols.push({ title: '下行CQI', dataIndex: 'dlcqi', width: 80, align: 'right' });
    }

    cols.push({
      title: '上行MCS',
      dataIndex: 'ulmcs',
      width: 80,
      align: 'right',
    });

    // MCS 列 — 436Q 拆分为 P/S 双载波
    if (use436Q) {
      cols.push(
        { title: 'P_Dlmcs', dataIndex: 'pDlmcs', width: 80, align: 'right' },
        { title: 'S_Dlmcs', dataIndex: 'sDlmcs', width: 80, align: 'right' },
      );
    } else {
      cols.push({ title: '下行MCS', dataIndex: 'dlmcs', width: 80, align: 'right' });
    }

    cols.push(
      { title: '发送功率(dBm)', dataIndex: 'txpower', width: 110, align: 'right' },
      { title: '上行BLER(%)', dataIndex: 'uplinkBler', width: 100, align: 'right' },
    );

    // BLER 列 — 436Q 拆分为 P_TB1/P_TB2/S_TB1/S_TB2
    if (use436Q) {
      cols.push(
        { title: 'P_TB1下行BLER(%)', dataIndex: 'p1DownlinkBler', width: 140, align: 'right' },
        { title: 'P_TB2下行BLER(%)', dataIndex: 'p2DownlinkBler', width: 140, align: 'right' },
        { title: 'S_TB1下行BLER(%)', dataIndex: 's1DownlinkBler', width: 140, align: 'right' },
        { title: 'S_TB2下行BLER(%)', dataIndex: 's2DownlinkBler', width: 140, align: 'right' },
      );
    } else {
      cols.push({ title: '下行BLER(%)', dataIndex: 'downlinkBler', width: 110, align: 'right' });
    }

    cols.push(
      { title: '路径损耗(dBm)', dataIndex: 'pathloss', width: 120, align: 'right' },
      { title: 'UE_S1AP_ID', dataIndex: 'ueS1apId', width: 110, align: 'right' },
      { title: 'MME_S1AP_ID', dataIndex: 'mmeS1apId', width: 120, align: 'right' },
    );

    return cols;
  }, [use436Q]);

  const mockData = useMemo(() => {
    if (!device) return [];
    const count = device.ueCount ?? 0;
    return generateMockUeData(Math.min(count, 50), use436Q);
  }, [device, use436Q]);

  return (
    <Drawer
      title={
        <span>
          UE 详情
          {device && (
            <>
              {' — '}
              {device.name || device.sn}
              {use436Q && <Tag color="blue" style={{ marginLeft: 8 }}>436Q</Tag>}
            </>
          )}
        </span>
      }
      placement="right"
      width={Math.min(window.innerWidth * 0.75, 1200)}
      open={!!device}
      onClose={onClose}
      destroyOnClose
    >
      <Table<UeRecord>
        columns={columns}
        dataSource={mockData}
        size="small"
        scroll={{ x: use436Q ? 2200 : 1600 }}
        pagination={mockData.length > 20 ? { pageSize: 20, showSizeChanger: false } : false}
      />
    </Drawer>
  );
}
