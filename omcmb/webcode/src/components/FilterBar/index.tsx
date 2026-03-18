import React, { useCallback, useEffect, useState } from 'react';
import { useT } from '@/hooks/useT';
import {
  Button,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  TreeSelect,
} from 'antd';
import {
  DownOutlined,
  ReloadOutlined,
  SearchOutlined,
  UpOutlined,
} from '@ant-design/icons';
import styles from './FilterBar.module.css';

const { RangePicker } = DatePicker;

export interface FilterField {
  name: string;
  label: string;
  type: 'input' | 'select' | 'date-range' | 'tree-select' | 'multi-select' | 'number';
  placeholder?: string;
  options?: { label: string; value: string | number }[];
  treeData?: { title: string; value: string; children?: unknown[] }[];
  span?: number;
  defaultValue?: unknown;
}

export interface FilterBarProps {
  filterId: string;
  fields: FilterField[];
  onSearch: (values: Record<string, unknown>) => void;
  onReset: () => void;
  collapsedRows?: number;
  extra?: React.ReactNode;
}

const COLS_PER_ROW = 6;
const COL_SPAN = 24 / COLS_PER_ROW; // 4

const SESSION_PREFIX = 'omc_filter_';

const FilterBar: React.FC<FilterBarProps> = ({
  filterId,
  fields,
  onSearch,
  onReset,
  collapsedRows = 1,
  extra,
}) => {
  const t = useT();
  const [form] = Form.useForm<Record<string, unknown>>();
  const [expanded, setExpanded] = useState(false);
  const storageKey = `${SESSION_PREFIX}${filterId}`;

  // Restore from sessionStorage on mount
  useEffect(() => {
    try {
      const stored = sessionStorage.getItem(storageKey);
      if (stored) {
        const parsed = JSON.parse(stored) as Record<string, unknown>;
        form.setFieldsValue(parsed);
      }
    } catch {
      // ignore
    }
  }, [form, storageKey]);

  const handleSearch = useCallback(() => {
    const values = form.getFieldsValue() as Record<string, unknown>;
    try {
      sessionStorage.setItem(storageKey, JSON.stringify(values));
    } catch {
      // ignore
    }
    onSearch(values);
  }, [form, onSearch, storageKey]);

  const handleReset = useCallback(() => {
    form.resetFields();
    try {
      sessionStorage.removeItem(storageKey);
    } catch {
      // ignore
    }
    onReset();
  }, [form, onReset, storageKey]);

  // Separate the first "input" field as the primary search field
  const searchField = fields.find((f) => f.type === 'input');
  const filterFields = fields.filter((f) => f !== searchField);

  // Calculate visible filter fields based on collapsed state
  const maxFiltersVisible = collapsedRows * COLS_PER_ROW - 1; // reserve 1 slot for actions
  const filtersToShow = expanded ? filterFields : filterFields.slice(0, maxFiltersVisible);
  const hasMore = filterFields.length > maxFiltersVisible;

  const renderField = (field: FilterField): React.ReactNode => {
    switch (field.type) {
      case 'input':
        return (
          <Input
            placeholder={field.placeholder ?? t('common.placeholder')}
            allowClear
          />
        );
      case 'select':
        return (
          <Select
            placeholder={field.placeholder ?? t('common.pleaseSelect')}
            allowClear
            options={field.options}
            style={{ width: '100%' }}
          />
        );
      case 'multi-select':
        return (
          <Select
            placeholder={field.placeholder ?? t('common.pleaseSelect')}
            mode="multiple"
            allowClear
            options={field.options}
            style={{ width: '100%' }}
            maxTagCount="responsive"
          />
        );
      case 'date-range':
        return (
          <RangePicker
            style={{ width: '100%' }}
            placeholder={[t('dateRange.start'), t('dateRange.end')]}
          />
        );
      case 'tree-select':
        return (
          <TreeSelect
            placeholder={field.placeholder ?? t('common.pleaseSelect')}
            allowClear
            treeData={field.treeData as Parameters<typeof TreeSelect>[0]['treeData']}
            style={{ width: '100%' }}
            treeCheckable
            showCheckedStrategy={TreeSelect.SHOW_PARENT}
          />
        );
      case 'number':
        return (
          <InputNumber
            placeholder={field.placeholder ?? t('common.placeholder')}
            style={{ width: '100%' }}
          />
        );
      default:
        return <Input placeholder={field.placeholder} allowClear />;
    }
  };

  const getFieldSpan = (field: FilterField): number => {
    const fieldSpan = field.span ?? 1;
    return fieldSpan * COL_SPAN;
  };

  return (
    <div className={styles.filterBarWrapper}>
      <Form form={form} size="small" layout="vertical">

        {/* 主搜索行：搜索框 + 查询/重置按钮 */}
        <div className={styles.searchRow}>
          {searchField && (
            <Form.Item
              name={searchField.name}
              initialValue={searchField.defaultValue}
              noStyle
            >
              <Input
                className={styles.searchInput}
                placeholder={searchField.placeholder ?? t('common.placeholder')}
                prefix={<SearchOutlined style={{ color: 'rgba(0,0,0,0.25)' }} />}
                allowClear
                onPressEnter={handleSearch}
              />
            </Form.Item>
          )}
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            {t('filter.query')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {t('common.reset')}
          </Button>
          {extra && <span className={styles.extraWrapper}>{extra}</span>}
        </div>

        {/* 筛选条件区 */}
        {filterFields.length > 0 && (
          <div className={styles.filterSection}>
            <Row gutter={[12, 0]} align="bottom">
              {filtersToShow.map((field) => (
                <Col key={field.name} span={getFieldSpan(field)}>
                  <Form.Item
                    name={field.name}
                    label={field.label}
                    initialValue={field.defaultValue}
                    className={styles.compactFormItem}
                    style={{ marginBottom: 12 }}
                  >
                    {renderField(field)}
                  </Form.Item>
                </Col>
              ))}

              {/* 展开/收起按钮 */}
              {hasMore && (
                <Col span={COL_SPAN} style={{ marginBottom: 12, display: 'flex', alignItems: 'flex-end', paddingBottom: 4 }}>
                  <Button
                    type="link"
                    size="small"
                    className={styles.expandBtn}
                    onClick={() => setExpanded(!expanded)}
                    icon={expanded ? <UpOutlined /> : <DownOutlined />}
                  >
                    {expanded ? t('filter.collapse') : t('filter.expand')}
                  </Button>
                </Col>
              )}
            </Row>
          </div>
        )}
      </Form>
    </div>
  );
};

export default FilterBar;
