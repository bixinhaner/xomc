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
import { DownOutlined, SearchOutlined, UpOutlined } from '@ant-design/icons';
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

  // Calculate which fields to show based on collapsed state
  const maxColsVisible = collapsedRows * COLS_PER_ROW;
  // Reserve last slot in last row for action buttons
  const fieldsToShow = expanded ? fields : fields.slice(0, maxColsVisible - 1);
  const hasMore = fields.length > maxColsVisible - 1;

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

  // Action buttons occupy one "cell"
  const actionSpan = COL_SPAN;

  return (
    <div className={styles.filterBarWrapper}>
      <Form form={form} layout="vertical" size="small">
        <Row gutter={[12, 0]} align="bottom">
          {fieldsToShow.map((field) => (
            <Col key={field.name} span={getFieldSpan(field)}>
              <Form.Item
                name={field.name}
                label={field.label}
                initialValue={field.defaultValue}
                className={styles.formItemWrapper}
                style={{ marginBottom: 16 }}
              >
                {renderField(field)}
              </Form.Item>
            </Col>
          ))}

          {/* Action column */}
          <Col
            span={actionSpan}
            className={styles.actionCol}
            style={{ marginBottom: 16 }}
          >
            {hasMore && (
              <Button
                type="link"
                size="small"
                className={styles.expandLink}
                onClick={() => setExpanded(!expanded)}
                icon={expanded ? <UpOutlined /> : <DownOutlined />}
              >
                {expanded ? t('filter.collapse') : t('filter.expand')}
              </Button>
            )}
            {extra && <span className={styles.extraWrapper}>{extra}</span>}
            <Button onClick={handleReset}>{t('common.reset')}</Button>
            <Button
              type="primary"
              icon={<SearchOutlined />}
              onClick={handleSearch}
            >
              {t('filter.query')}
            </Button>
          </Col>
        </Row>
      </Form>
    </div>
  );
};

export default FilterBar;
