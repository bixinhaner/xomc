import { useState } from 'react';
import { Button, Pagination, Space, Typography } from 'antd';

import type { TranslateFn } from '@/hooks/useT';
import {
  MML_MAX_TASK_DEVICES,
  MML_PREVIEW_PAGE_SIZE,
  MML_PREVIEW_PAGE_SIZE_OPTIONS,
} from '@core/utils/mmlTaskScale';

interface PaginatedDeviceSnListProps {
  deviceSns: string[];
  onChange: (deviceSns: string[]) => void;
  t: TranslateFn;
}

export default function PaginatedDeviceSnList({
  deviceSns,
  onChange,
  t,
}: PaginatedDeviceSnListProps) {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(MML_PREVIEW_PAGE_SIZE);
  const pageCount = Math.max(1, Math.ceil(deviceSns.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const startIndex = (currentPage - 1) * pageSize;
  const visibleSns = deviceSns.slice(startIndex, startIndex + pageSize);
  const summary = t('mml.selectedDeviceSummary', {
    count: deviceSns.length,
    max: MML_MAX_TASK_DEVICES,
  });
  const end = Math.min(startIndex + visibleSns.length, deviceSns.length);

  if (deviceSns.length === 0) return null;

  return (
    <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Typography.Text strong>{summary}</Typography.Text>
        <Button type="link" size="small" onClick={() => onChange([])}>
          {t('mml.clearSelectedDevices')}
        </Button>
      </Space>
      {deviceSns.length > MML_MAX_TASK_DEVICES ? (
        <Typography.Text type="danger">
          {t('mml.taskDeviceLimitExceeded', {
            current: deviceSns.length,
            max: MML_MAX_TASK_DEVICES,
          })}
        </Typography.Text>
      ) : null}
      <ul
        aria-label={summary}
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
          gap: 6,
          margin: 0,
          padding: 0,
          listStyle: 'none',
        }}
      >
        {visibleSns.map((sn) => (
          <li
            key={sn}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              minWidth: 0,
              padding: '5px 8px',
              border: '1px solid #f0f0f0',
              borderRadius: 6,
            }}
          >
            <Typography.Text ellipsis style={{ maxWidth: 170 }} title={sn}>
              {sn}
            </Typography.Text>
            <Button
              type="text"
              danger
              size="small"
              aria-label={t('mml.removeSelectedDevice', { sn })}
              onClick={() => onChange(deviceSns.filter((value) => value !== sn))}
            >
              ×
            </Button>
          </li>
        ))}
      </ul>
      <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {t('mml.deviceRange', { start: startIndex + 1, end, total: deviceSns.length })}
        </Typography.Text>
        <Pagination
          current={currentPage}
          pageSize={pageSize}
          total={deviceSns.length}
          size="small"
          showSizeChanger
          pageSizeOptions={MML_PREVIEW_PAGE_SIZE_OPTIONS.map(String)}
          onChange={(nextPage, nextPageSize) => {
            setPage(nextPageSize === pageSize ? nextPage : 1);
            setPageSize(nextPageSize);
          }}
        />
      </Space>
    </Space>
  );
}
