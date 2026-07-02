import React, { useCallback, useEffect, useState } from 'react';
import dayjs from 'dayjs';
import { useT } from '@/hooks/useT';
import {
  Button,
  DatePicker,
  Form,
  Input,
  InputNumber,
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
  showTime?: boolean;
  options?: { label: string; value: string | number }[];
  treeData?: { title: string; value: string; children?: unknown[] }[];
  span?: number;
  /** 固定宽度字段。设置后不再按 flex:1 拉伸。 */
  width?: number;
  /** 覆盖默认 min-width（140px）。传较小数值可让筛选框更紧凑（如类型/状态筛选） */
  minWidth?: number;
  defaultValue?: unknown;
}

export interface FilterBarProps {
  filterId: string;
  fields: FilterField[];
  onSearch: (values: Record<string, unknown>) => void;
  onReset: () => void;
  collapsedRows?: number;
  extra?: React.ReactNode;
  /** @deprecated 不再需要，FilterBar 默认无边框无背景 */
  noDefaultStyle?: boolean;
  /** 初始值（优先级高于 sessionStorage），传空对象 {} 表示清空所有筛选 */
  initialValues?: Record<string, unknown>;
}

const FIELDS_PER_ROW = 6;

const SESSION_PREFIX = 'omc_filter_';

function hydrateFormValues(
  values: Record<string, unknown>,
  fields: FilterField[]
): Record<string, unknown> {
  const hydratedValues = { ...values };

  fields.forEach((field) => {
    if (field.type !== 'date-range') {
      return;
    }

    const rawValue = hydratedValues[field.name];
    if (!Array.isArray(rawValue)) {
      return;
    }

    hydratedValues[field.name] = rawValue.map((item) => {
      if (item == null || dayjs.isDayjs(item)) {
        return item;
      }

      const parsed = dayjs(item as string | number | Date);
      return parsed.isValid() ? parsed : item;
    });
  });

  return hydratedValues;
}

function serializeFormValues(
  values: Record<string, unknown>,
  fields: FilterField[]
): Record<string, unknown> {
  const serializedValues = { ...values };

  fields.forEach((field) => {
    if (field.type !== 'date-range') {
      return;
    }

    const rawValue = serializedValues[field.name];
    if (!Array.isArray(rawValue)) {
      return;
    }

    serializedValues[field.name] = rawValue.map((item) => {
      if (item == null) {
        return item;
      }

      if (dayjs.isDayjs(item)) {
        return item.toISOString();
      }

      return item;
    });
  });

  return serializedValues;
}

const FilterBar: React.FC<FilterBarProps> = ({
  filterId,
  fields,
  onSearch,
  onReset,
  collapsedRows = 1,
  extra,
  noDefaultStyle: _noDefaultStyle,
  initialValues,
}) => {
  const t = useT();
  const [form] = Form.useForm<Record<string, unknown>>();
  const [expanded, setExpanded] = useState(false);
  const storageKey = `${SESSION_PREFIX}${filterId}`;
  const fieldsRef = React.useRef(fields);
  const initialValuesSignature = React.useMemo(() => {
    if (initialValues === undefined) {
      return '__undefined__';
    }

    const entries = Object.entries(initialValues).sort(([left], [right]) => left.localeCompare(right));
    return JSON.stringify(entries);
  }, [initialValues]);

  useEffect(() => {
    fieldsRef.current = fields;
  }, [fields]);

  // Restore from initialValues (priority) or sessionStorage on mount
  // Also sync form when initialValues changes (e.g., URL params cleared)
  useEffect(() => {
    const currentFields = fieldsRef.current;

    // 如果传入了 initialValues
    if (initialValues !== undefined) {
      if (Object.keys(initialValues).length === 0) {
        // 传入空对象时，明确清空所有表单字段
        const resetValues: Record<string, undefined> = {};
        currentFields.forEach((field) => {
          resetValues[field.name] = undefined;
        });
        form.setFieldsValue(resetValues as Parameters<typeof form.setFieldsValue>[0]);
      } else {
        // 非空 initialValues，直接设置
        form.setFieldsValue(
          hydrateFormValues(initialValues, currentFields) as Parameters<typeof form.setFieldsValue>[0]
        );
      }
      return;
    }
    // 如果没有传 initialValues，从 sessionStorage 恢复
    try {
      const stored = sessionStorage.getItem(storageKey);
      if (stored) {
        const parsed = JSON.parse(stored) as Record<string, unknown>;
        form.setFieldsValue(
          hydrateFormValues(parsed, currentFields) as Parameters<typeof form.setFieldsValue>[0]
        );
      }
    } catch {
      // ignore
    }
  }, [form, storageKey, initialValuesSignature]);

  const handleSearch = useCallback(() => {
    const values = serializeFormValues(
      form.getFieldsValue() as Record<string, unknown>,
      fields
    );
    try {
      sessionStorage.setItem(storageKey, JSON.stringify(values));
    } catch {
      // ignore
    }
    onSearch(values);
  }, [fields, form, onSearch, storageKey]);

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
  const maxFieldsVisible = collapsedRows * FIELDS_PER_ROW;
  const fieldsToShow = expanded ? fields : fields.slice(0, maxFieldsVisible - 1);
  const hasMore = fields.length > maxFieldsVisible - 1;

  const renderField = (field: FilterField): React.ReactNode => {
    switch (field.type) {
      case 'input':
        return (
          <Input
            placeholder={field.placeholder ?? field.label}
            allowClear
            onPressEnter={handleSearch}
          />
        );
      case 'select':
        return (
          <Select
            placeholder={field.placeholder ?? field.label}
            allowClear
            options={field.options}
            style={{ width: '100%' }}
          />
        );
      case 'multi-select':
        return (
          <Select
            placeholder={field.placeholder ?? field.label}
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
            showTime={field.showTime}
            format={field.showTime ? 'YYYY-MM-DD HH:mm:ss' : undefined}
            placeholder={[t('dateRange.start'), t('dateRange.end')]}
          />
        );
      case 'tree-select':
        return (
          <TreeSelect
            placeholder={field.placeholder ?? field.label}
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
            placeholder={field.placeholder ?? field.label}
            style={{ width: '100%' }}
          />
        );
      default:
        return <Input placeholder={field.placeholder ?? field.label} allowClear />;
    }
  };

  return (
    <div className={styles.filterBarWrapper}>
      <Form form={form} size="small">
        <div className={styles.fieldsRow}>
          {fieldsToShow.map((field) => (
            <div
              key={field.name}
              className={styles.fieldItem}
              // 2026-06-03:字段较窄时 placeholder 会被截断;hover 显示完整提示文案(原生 title)。
              title={field.placeholder ?? field.label}
              style={{
                ...(field.width !== undefined
                  ? {
                      flex: `0 0 ${field.width}px`,
                      width: field.width,
                      minWidth: field.width,
                      maxWidth: field.width,
                    }
                  : {
                      flex: field.span ?? 1,
                      ...(field.minWidth !== undefined ? { minWidth: field.minWidth } : {}),
                    }),
              }}
            >
              <Form.Item
                name={field.name}
                initialValue={field.defaultValue}
                className={styles.formItem}
              >
                {renderField(field)}
              </Form.Item>
            </div>
          ))}

          {/* Action area */}
          <div className={styles.actionArea}>
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
            <a className={styles.resetLink} onClick={handleReset}>
              {t('common.reset')}
            </a>
            <button
              className={styles.searchBtn}
              onClick={handleSearch}
              type="button"
              title={t('filter.query')}
            >
              <SearchOutlined />
            </button>
            {/* extra 放在搜索按钮之后（to-do-list #4：+新增 按钮紧贴搜索按钮） */}
            {extra && <span className={styles.extraWrapper}>{extra}</span>}
          </div>
        </div>
      </Form>
    </div>
  );
};

export default FilterBar;
