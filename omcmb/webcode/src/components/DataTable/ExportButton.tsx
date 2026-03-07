import React from 'react';
import { Button, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { DownloadOutlined, FileExcelOutlined, FileTextOutlined } from '@ant-design/icons';

interface ExportButtonProps {
  onExport: (format: 'xlsx' | 'csv') => void;
}

const ExportButton: React.FC<ExportButtonProps> = ({ onExport }) => {
  const items: MenuProps['items'] = [
    {
      key: 'xlsx',
      icon: <FileExcelOutlined style={{ color: '#52C41A' }} />,
      label: '导出 Excel',
      onClick: () => onExport('xlsx'),
    },
    {
      key: 'csv',
      icon: <FileTextOutlined style={{ color: 'var(--color-primary-600)' }} />,
      label: '导出 CSV',
      onClick: () => onExport('csv'),
    },
  ];

  return (
    <Dropdown menu={{ items }} placement="bottomRight" trigger={['click']}>
      <Button icon={<DownloadOutlined />} size="small" title="导出">
        导出
      </Button>
    </Dropdown>
  );
};

export default ExportButton;
