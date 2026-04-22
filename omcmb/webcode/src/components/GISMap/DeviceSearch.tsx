/**
 * 设备搜索组件
 * @module components/GISMap/DeviceSearch
 */

import React, { useState, useRef, useEffect } from 'react';
import { Spin, Empty, Typography } from 'antd';
import { CloseOutlined, DownOutlined } from '@ant-design/icons';
import { useDebounce } from 'ahooks';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';
import type { DeviceSearchResult } from '@core/types/map';
import { DEVICE_STATUS_CONFIG } from './constants';
import styles from './styles.module.css';

const { Text } = Typography;

interface DeviceSearchProps {
  /** 搜索关键词 */
  value?: string;
  /** 关键词变化回调 */
  onChange: (keyword: string) => void;
  /** 搜索结果 */
  results?: DeviceSearchResult[];
  /** 结果项点击回调 */
  onResultClick: (result: DeviceSearchResult) => void;
  /** 加载状态 */
  loading?: boolean;
  /** 是否展开结果列表 */
  expanded?: boolean;
}

/**
 * 设备搜索组件
 * 根据 UI 设计图 GISMap_UI_Design_Main.svg
 * - 搜索框
 * - 结果列表（名称 | SN | 状态）
 * - 点击定位高亮
 */
const DeviceSearch: React.FC<DeviceSearchProps> = ({
  value = '',
  onChange,
  results = [],
  onResultClick,
  loading = false,
  expanded: controlledExpanded,
}) => {
  const t = useT();
  const token = useThemeToken();
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const [internalExpanded, setInternalExpanded] = useState(false);
  const expanded = controlledExpanded ?? internalExpanded;
  const [highlightedIndex, setHighlightedIndex] = useState(-1);

  const debouncedValue = useDebounce(value, { wait: 300 });

  // 点击外部关闭
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setInternalExpanded(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // 自动展开结果列表
  useEffect(() => {
    if (debouncedValue.length >= 2 && results.length > 0) {
      setInternalExpanded(true);
    }
  }, [debouncedValue, results.length]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    onChange(newValue);
    if (newValue.length >= 2) {
      setInternalExpanded(true);
    } else {
      setInternalExpanded(false);
    }
    setHighlightedIndex(-1);
  };

  const handleClear = () => {
    onChange('');
    setInternalExpanded(false);
    setHighlightedIndex(-1);
    inputRef.current?.focus();
  };

  const handleResultClick = (result: DeviceSearchResult) => {
    onResultClick(result);
    setInternalExpanded(false);
    setHighlightedIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!expanded || results.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setHighlightedIndex(prev =>
          prev < results.length - 1 ? prev + 1 : 0
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setHighlightedIndex(prev =>
          prev > 0 ? prev - 1 : results.length - 1
        );
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0 && highlightedIndex < results.length) {
          handleResultClick(results[highlightedIndex]);
        }
        break;
      case 'Escape':
        setInternalExpanded(false);
        setHighlightedIndex(-1);
        break;
    }
  };

  const containerStyle: React.CSSProperties = {
    position: 'absolute',
    left: 300,
    top: 20,
    zIndex: 500,
    width: 320,
  };

  const searchBoxStyle: React.CSSProperties = {
    background: token.colorBgContainer,
    borderRadius: 12,
    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
    border: `2px solid ${expanded ? token.colorPrimary : token.colorBorderSecondary}`,
    overflow: 'hidden',
    transition: 'border-color 0.2s',
  };

  const inputContainerStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    padding: '0 16px',
    height: 48,
    gap: 12,
  };

  const resultListStyle: React.CSSProperties = {
    maxHeight: 320,
    overflow: 'auto',
    borderTop: `1px solid ${token.colorBorderSecondary}`,
  };

  const resultItemStyle = (isHighlighted: boolean): React.CSSProperties => ({
    padding: '12px 16px',
    cursor: 'pointer',
    background: isHighlighted ? token.colorPrimaryBg : 'transparent',
    transition: 'background 0.15s',
  });

  const footerStyle: React.CSSProperties = {
    padding: '10px 16px',
    borderTop: `1px solid ${token.colorBorderSecondary}`,
    textAlign: 'center',
  };

  return (
    <div ref={containerRef} style={containerStyle} className={styles.deviceSearch}>
      <div style={searchBoxStyle}>
        {/* 搜索框 */}
        <div style={inputContainerStyle}>
          {/* 搜索图标 */}
          <div style={{ position: 'relative', width: 16, height: 16 }}>
            <div
              style={{
                width: 12,
                height: 12,
                border: `1px solid ${token.colorPrimary}`,
                borderRadius: '50%',
              }}
            />
            <div
              style={{
                position: 'absolute',
                right: -2,
                bottom: -2,
                width: 6,
                height: 2,
                background: token.colorPrimary,
                transform: 'rotate(45deg)',
              }}
            />
          </div>

          {/* 输入框 */}
          <input
            ref={inputRef as any}
            type="text"
            value={value}
            onChange={handleInputChange}
            onFocus={() => {
              if (value.length >= 2 && results.length > 0) {
                setInternalExpanded(true);
              }
            }}
            onKeyDown={handleKeyDown}
            placeholder={t('device.searchPlaceholder')}
            style={{
              flex: 1,
              border: 'none',
              outline: 'none',
              fontSize: 13,
              color: token.colorText,
              background: 'transparent',
            }}
          />

          {/* 清除按钮 */}
          {value && (
            <div
              onClick={handleClear}
              style={{
                width: 24,
                height: 20,
                background: token.colorBgLayout,
                borderRadius: 4,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
              }}
            >
              <CloseOutlined style={{ fontSize: 12, color: token.colorTextSecondary }} />
            </div>
          )}

          {/* 展开箭头 */}
          <DownOutlined
            style={{
              fontSize: 12,
              color: token.colorPrimary,
              transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
              transition: 'transform 0.2s',
            }}
          />
        </div>

        {/* 结果列表 */}
        {expanded && (
          <div style={resultListStyle}>
            {loading ? (
              <div style={{ textAlign: 'center', padding: 24 }}>
                <Spin size="small" />
              </div>
            ) : results.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  <div>
                    <Text style={{ fontSize: 13, color: token.colorTextSecondary }}>
                      🔍
                    </Text>
                    <br />
                    <Text style={{ fontSize: 12, color: token.colorTextSecondary }}>
                      {t('device.noSearchResults')}
                    </Text>
                    <br />
                    <Text style={{ fontSize: 11, color: token.colorTextDisabled }}>
                      {t('device.tryOtherKeywords')}
                    </Text>
                  </div>
                }
                style={{ padding: '24px 0' }}
              />
            ) : (
              <>
                {results.map((result, index) => {
                  const config = DEVICE_STATUS_CONFIG[result.status] || DEVICE_STATUS_CONFIG.offline;
                  const isHighlighted = index === highlightedIndex;

                  return (
                    <div
                      key={result.id}
                      style={resultItemStyle(isHighlighted)}
                      onClick={() => handleResultClick(result)}
                      onMouseEnter={() => setHighlightedIndex(index)}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        {/* 状态点 */}
                        <div
                          style={{
                            width: 12,
                            height: 12,
                            borderRadius: '50%',
                            background: config.color,
                            flexShrink: 0,
                          }}
                        />

                        {/* 名称 */}
                        <Text
                          strong
                          style={{ fontSize: 13, color: token.colorText }}
                        >
                          {result.name}
                        </Text>
                      </div>

                      <div
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                          marginTop: 4,
                          paddingLeft: 20,
                        }}
                      >
                        <Text style={{ fontSize: 11, color: token.colorTextSecondary }}>
                          SN: {result.sn}
                        </Text>
                        <Text style={{ fontSize: 11, color: config.color }}>
                          {config.text}
                        </Text>
                      </div>
                    </div>
                  );
                })}

                {/* 结果计数 */}
                <div style={footerStyle}>
                  <Text style={{ fontSize: 11, color: token.colorTextSecondary }}>
                    {t('device.searchResultsCount', { count: results.length })}
                  </Text>
                </div>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default DeviceSearch;
