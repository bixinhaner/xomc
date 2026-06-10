import React, { useMemo, useState } from 'react';
import { Button, Checkbox, Input, Space, Typography } from 'antd';
import {
  DoubleLeftOutlined,
  DoubleRightOutlined,
  LeftOutlined,
  RightOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useThemeToken } from '@/hooks/useThemeToken';

export interface TransferItem {
  key: string;
  label: string;
  description?: string;
  disabled?: boolean;
}

export interface TransferBoxProps {
  dataSource: TransferItem[];
  targetKeys: string[];
  onChange: (keys: string[]) => void;
  titles?: [string, string];
  searchable?: boolean;
  maxTarget?: number;
  style?: React.CSSProperties;
}

const TransferBox: React.FC<TransferBoxProps> = ({
  dataSource,
  targetKeys,
  onChange,
  titles = ['可选项', '已选项'],
  searchable = true,
  maxTarget,
  style,
}) => {
  const [leftSearch, setLeftSearch] = useState('');
  const [rightSearch, setRightSearch] = useState('');
  const [leftChecked, setLeftChecked] = useState<string[]>([]);
  const [rightChecked, setRightChecked] = useState<string[]>([]);
  const token = useThemeToken();

  const leftItems = useMemo(
    () =>
      dataSource
        .filter((item) => !targetKeys.includes(item.key))
        .filter(
          (item) =>
            !leftSearch ||
            item.label.toLowerCase().includes(leftSearch.toLowerCase())
        ),
    [dataSource, targetKeys, leftSearch]
  );

  const rightItems = useMemo(
    () =>
      dataSource
        .filter((item) => targetKeys.includes(item.key))
        .filter(
          (item) =>
            !rightSearch ||
            item.label.toLowerCase().includes(rightSearch.toLowerCase())
        ),
    [dataSource, targetKeys, rightSearch]
  );

  const moveRight = (keys: string[]) => {
    const toAdd = keys.filter(
      (k) => !targetKeys.includes(k) && !dataSource.find((i) => i.key === k)?.disabled
    );
    const next = [...targetKeys, ...toAdd];
    if (maxTarget && next.length > maxTarget) return;
    onChange(next);
    setLeftChecked([]);
  };

  const moveLeft = (keys: string[]) => {
    onChange(targetKeys.filter((k) => !keys.includes(k)));
    setRightChecked([]);
  };

  const moveAllRight = () => {
    const toAdd = leftItems.filter((i) => !i.disabled).map((i) => i.key);
    const next = [...new Set([...targetKeys, ...toAdd])];
    if (maxTarget && next.length > maxTarget) return;
    onChange(next);
    setLeftChecked([]);
  };

  const moveAllLeft = () => {
    const toRemove = new Set(rightItems.filter((i) => !i.disabled).map((i) => i.key));
    onChange(targetKeys.filter((k) => !toRemove.has(k)));
    setRightChecked([]);
  };

  const toggleLeftCheck = (key: string, checked: boolean) => {
    setLeftChecked((prev) => (checked ? [...prev, key] : prev.filter((k) => k !== key)));
  };

  const toggleRightCheck = (key: string, checked: boolean) => {
    setRightChecked((prev) => (checked ? [...prev, key] : prev.filter((k) => k !== key)));
  };

  const panelStyle: React.CSSProperties = {
    border: `1px solid ${token.colorBorder}`,
    borderRadius: 6,
    display: 'flex',
    flexDirection: 'column',
    flex: 1,
    minWidth: 0,
    overflow: 'hidden',
  };

  const headerStyle: React.CSSProperties = {
    padding: '8px 12px',
    background: token.colorBgLayout,
    borderBottom: `1px solid ${token.colorBorderSecondary}`,
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  };

  const itemStyle = (checked: boolean, disabled: boolean): React.CSSProperties => ({
    display: 'flex',
    alignItems: 'flex-start',
    gap: 8,
    padding: '6px 12px',
    background: checked ? token.controlItemBgActive : 'transparent',
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.5 : 1,
    transition: 'background 0.15s',
    borderBottom: `1px solid ${token.colorBorderSecondary}`,
  });

  const renderPanel = (
    items: TransferItem[],
    title: string,
    count: number,
    searchVal: string,
    setSearch: (v: string) => void,
    checkedKeys: string[],
    toggleCheck: (key: string, checked: boolean) => void,
    toggleAll: () => void
  ) => {
    const allChecked = items.length > 0 && items.filter((i) => !i.disabled).every((i) => checkedKeys.includes(i.key));
    const someChecked = items.some((i) => checkedKeys.includes(i.key));

    return (
      <div style={panelStyle}>
        <div style={headerStyle}>
          <Checkbox
            checked={allChecked}
            indeterminate={!allChecked && someChecked}
            onChange={(e) => {
              if (e.target.checked) {
                toggleAll();
              } else {
                // uncheck all visible
                const visibleKeys = items.filter((i) => !i.disabled).map((i) => i.key);
                visibleKeys.forEach((k) => toggleCheck(k, false));
              }
            }}
          />
          <span style={{ fontWeight: 500, fontSize: 13, flex: 1 }}>
            {title}
          </span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {checkedKeys.length > 0 ? `${checkedKeys.length}/` : ''}{count} 项
          </Typography.Text>
        </div>

        {searchable && (
          <div style={{ padding: '6px 8px', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
            <Input
              prefix={<SearchOutlined style={{ color: token.colorTextDisabled }} />}
              placeholder="搜索..."
              value={searchVal}
              onChange={(e) => setSearch(e.target.value)}
              size="small"
              allowClear
            />
          </div>
        )}

        <div style={{ flex: 1, overflow: 'auto' }}>
          {items.length === 0 ? (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                height: '100%',
                color: token.colorTextDisabled,
                fontSize: 13,
                padding: 24,
              }}
            >
              暂无数据
            </div>
          ) : (
            items.map((item) => {
              const checked = checkedKeys.includes(item.key);
              return (
                <div
                  key={item.key}
                  style={itemStyle(checked, item.disabled ?? false)}
                  onClick={() => {
                    if (!item.disabled) toggleCheck(item.key, !checked);
                  }}
                >
                  <Checkbox
                    checked={checked}
                    disabled={item.disabled}
                    onChange={(e) => {
                      e.stopPropagation();
                      if (!item.disabled) toggleCheck(item.key, e.target.checked);
                    }}
                    onClick={(e) => e.stopPropagation()}
                  />
                  <div>
                    <div style={{ fontSize: 13, color: token.colorTextHeading }}>{item.label}</div>
                    {item.description && (
                      <div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: 1 }}>
                        {item.description}
                      </div>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>
    );
  };

  return (
    <div style={{ display: 'flex', gap: 8, alignItems: 'stretch', ...style }}>
      {/* Left panel */}
      {renderPanel(
        leftItems,
        titles[0],
        leftItems.length,
        leftSearch,
        setLeftSearch,
        leftChecked,
        toggleLeftCheck,
        () => {
          const keys = leftItems.filter((i) => !i.disabled).map((i) => i.key);
          setLeftChecked(keys);
        }
      )}

      {/* Transfer buttons */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 8,
          flexShrink: 0,
        }}
      >
        <Space orientation="vertical" size={6}>
          <Button
            icon={<DoubleRightOutlined />}
            size="small"
            onClick={moveAllRight}
            disabled={leftItems.filter((i) => !i.disabled).length === 0}
            title="全部移入"
          />
          <Button
            icon={<RightOutlined />}
            size="small"
            type="primary"
            onClick={() => moveRight(leftChecked)}
            disabled={leftChecked.length === 0}
            title="移入"
          />
          <Button
            icon={<LeftOutlined />}
            size="small"
            onClick={() => moveLeft(rightChecked)}
            disabled={rightChecked.length === 0}
            title="移出"
          />
          <Button
            icon={<DoubleLeftOutlined />}
            size="small"
            onClick={moveAllLeft}
            disabled={rightItems.filter((i) => !i.disabled).length === 0}
            title="全部移出"
          />
        </Space>
      </div>

      {/* Right panel */}
      {renderPanel(
        rightItems,
        titles[1],
        rightItems.length,
        rightSearch,
        setRightSearch,
        rightChecked,
        toggleRightCheck,
        () => {
          const keys = rightItems.filter((i) => !i.disabled).map((i) => i.key);
          setRightChecked(keys);
        }
      )}
    </div>
  );
};

export default TransferBox;
