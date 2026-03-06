import React, { useState } from 'react';
import { Button, DatePicker, Input, Popover, Select, Space } from 'antd';
import { FilterOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

interface FilterOption {
  label: string;
  value: string;
}

interface ColumnFilterProps {
  columnKey: string;
  filterType?: 'text' | 'select' | 'date';
  filterOptions?: FilterOption[];
  value?: string;
  onFilter: (columnKey: string, value: string | undefined) => void;
}

const ColumnFilter: React.FC<ColumnFilterProps> = ({
  columnKey,
  filterType = 'text',
  filterOptions = [],
  value,
  onFilter,
}) => {
  const t = useT();
  const token = useThemeToken();
  const [inputValue, setInputValue] = useState(value ?? '');
  const [open, setOpen] = useState(false);

  const handleConfirm = () => {
    onFilter(columnKey, inputValue || undefined);
    setOpen(false);
  };

  const handleReset = () => {
    setInputValue('');
    onFilter(columnKey, undefined);
    setOpen(false);
  };

  const content = (
    <div style={{ width: 220 }}>
      {filterType === 'text' && (
        <Input
          placeholder={t('table.searchPlaceholder')}
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onPressEnter={handleConfirm}
          allowClear
          autoFocus
        />
      )}
      {filterType === 'select' && (
        <Select
          style={{ width: '100%' }}
          placeholder={t('common.pleaseSelect')}
          value={inputValue || undefined}
          onChange={(v: string) => setInputValue(v ?? '')}
          options={filterOptions}
          allowClear
        />
      )}
      {filterType === 'date' && (
        <DatePicker
          style={{ width: '100%' }}
          onChange={(_date, dateStr) =>
            setInputValue(Array.isArray(dateStr) ? dateStr[0] : dateStr)
          }
        />
      )}
      <Space style={{ marginTop: 8, width: '100%', justifyContent: 'flex-end' }}>
        <Button size="small" onClick={handleReset}>
          {t('common.reset')}
        </Button>
        <Button size="small" type="primary" onClick={handleConfirm}>
          {t('common.confirm')}
        </Button>
      </Space>
    </div>
  );

  const isFiltered = Boolean(value);

  return (
    <Popover
      content={content}
      trigger="click"
      open={open}
      onOpenChange={setOpen}
      placement="bottom"
    >
      <FilterOutlined
        style={{
          marginLeft: 4,
          fontSize: 12,
          color: isFiltered ? token.colorPrimary : '#bfbfbf',
          cursor: 'pointer',
        }}
        onClick={(e) => {
          e.stopPropagation();
          setOpen(!open);
        }}
      />
    </Popover>
  );
};

export default ColumnFilter;
