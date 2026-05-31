import { useMemo, useState, useCallback } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { App, Button, Form, Modal, Select, Space, Table, Typography } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import { adminApi } from '@core/services/api/adminApi';
import type {
  DictionarySource,
  DictionaryPreviewRow,
} from '@core/services/api/adminApi';
import { useT } from '@/hooks/useT';

/**
 * SourcePicker — 字典「数据源(可选)」折叠区域。
 *
 * T-0182:用户在「新增字典」/「编辑字典」对话框里选择
 *   ① 来源表(白名单 12 张)
 *   ② Label 字段
 *   ③ Value 字段
 *   ④ 可选「测试 → 预览前 10 条」dry-run
 *
 * 受控接口:
 *   - value:{ sourceTable, sourceLabelField, sourceValueField } —— 三字段同时空 = 未绑定
 *   - onChange:任一字段改动后调,父表单监听
 *   - disabled:编辑现有手工字典时 disabled,避免误绑(允许在 Modal 顶层放开关切换)
 *
 * 设计要点:
 *   - listDictionarySources 用 staleTime: Infinity,启动期加载后内存常驻;白名单不会变
 *   - sourceLabelField/sourceValueField 联动 sourceTable,切表时自动清空两个字段值
 *   - 选同字段时 antd Select 不需要特别处理,后端 v1 已允许 label==value(常见场景)
 *   - 「测试」按钮在三字段都填后启用;Modal 展示前 N 条 + 总数
 */
export interface SourcePickerValue {
  sourceTable?: string | null;
  sourceLabelField?: string | null;
  sourceValueField?: string | null;
}

interface SourcePickerProps {
  value?: SourcePickerValue;
  onChange?: (v: SourcePickerValue) => void;
  disabled?: boolean;
}

export default function SourcePicker({ value, onChange, disabled = false }: SourcePickerProps) {
  const t = useT();
  const { message } = App.useApp();

  const { data: sources = [], isLoading: sourcesLoading } = useQuery<DictionarySource[]>({
    queryKey: ['dict-sources'],
    queryFn: () => adminApi.listDictionarySources(),
    staleTime: Infinity,
  });

  // 当前选中的 table → spec(fields 列表),用于联动渲染 label/value 下拉
  const currentTableSpec = useMemo<DictionarySource | null>(
    () => sources.find((s) => s.table === value?.sourceTable) ?? null,
    [sources, value?.sourceTable],
  );

  const fieldOptions = useMemo(
    () =>
      (currentTableSpec?.fields ?? []).map((f) => ({
        value: f.column,
        label: `${f.display} (${f.column}, ${f.type})`,
      })),
    [currentTableSpec],
  );

  const tableOptions = useMemo(
    () =>
      sources.map((s) => ({
        value: s.table,
        label: `${s.display_name} (${s.table})`,
      })),
    [sources],
  );

  // 字段联动:切换 table 时清空 label/value(它们绑定的是旧表的字段名)
  const handleTableChange = useCallback(
    (next: string | null) => {
      onChange?.({
        sourceTable: next || null,
        sourceLabelField: null,
        sourceValueField: null,
      });
    },
    [onChange],
  );

  const handleLabelChange = useCallback(
    (next: string | null) => {
      onChange?.({
        sourceTable: value?.sourceTable ?? null,
        sourceLabelField: next || null,
        sourceValueField: value?.sourceValueField ?? null,
      });
    },
    [onChange, value],
  );

  const handleValueChange = useCallback(
    (next: string | null) => {
      onChange?.({
        sourceTable: value?.sourceTable ?? null,
        sourceLabelField: value?.sourceLabelField ?? null,
        sourceValueField: next || null,
      });
    },
    [onChange, value],
  );

  const previewMutation = useMutation({
    mutationFn: () =>
      adminApi.previewDictionarySource({
        table: value?.sourceTable as string,
        label: value?.sourceLabelField as string,
        value: value?.sourceValueField as string,
        limit: 10,
      }),
    onError: (err) => {
      const msg = err instanceof Error ? err.message : String(err);
      void message.error(t('dictionary.source.previewFailed', { error: msg }));
    },
  });

  const [previewOpen, setPreviewOpen] = useState(false);

  const canPreview =
    !disabled &&
    !!value?.sourceTable &&
    !!value?.sourceLabelField &&
    !!value?.sourceValueField;

  const handlePreview = useCallback(() => {
    setPreviewOpen(true);
    previewMutation.mutate();
  }, [previewMutation]);

  const previewColumns = useMemo(
    () => [
      {
        title: t('dictionary.label'),
        dataIndex: 'label',
        key: 'label',
        ellipsis: true,
      },
      {
        title: t('dictionary.value'),
        dataIndex: 'value',
        key: 'value',
        ellipsis: true,
      },
    ],
    [t],
  );

  return (
    <>
      <div style={{ padding: '8px 0 4px', borderTop: '1px dashed #f0f0f0', marginTop: 8 }}>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {t('dictionary.source.title')}
        </Typography.Text>
      </div>

      <Form.Item label={t('dictionary.source.table')}>
        <Select
          loading={sourcesLoading}
          options={tableOptions}
          value={value?.sourceTable ?? undefined}
          onChange={handleTableChange}
          placeholder={t('dictionary.source.tablePlaceholder')}
          allowClear
          showSearch
          optionFilterProp="label"
          disabled={disabled}
        />
      </Form.Item>

      <Form.Item label={t('dictionary.source.labelField')}>
        <Select
          options={fieldOptions}
          value={value?.sourceLabelField ?? undefined}
          onChange={handleLabelChange}
          placeholder={t('dictionary.source.labelPlaceholder')}
          allowClear
          showSearch
          optionFilterProp="label"
          disabled={disabled || !value?.sourceTable}
        />
      </Form.Item>

      <Form.Item label={t('dictionary.source.valueField')}>
        <Select
          options={fieldOptions}
          value={value?.sourceValueField ?? undefined}
          onChange={handleValueChange}
          placeholder={t('dictionary.source.valuePlaceholder')}
          allowClear
          showSearch
          optionFilterProp="label"
          disabled={disabled || !value?.sourceTable}
        />
      </Form.Item>

      <Form.Item>
        <Space>
          <Button
            icon={<EyeOutlined />}
            disabled={!canPreview}
            loading={previewMutation.isPending && previewOpen}
            onClick={handlePreview}
            size="small"
          >
            {t('dictionary.source.previewBtn')}
          </Button>
          {value?.sourceTable && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {t('dictionary.source.managedHint')}
            </Typography.Text>
          )}
        </Space>
      </Form.Item>

      <Modal
        title={t('dictionary.source.previewTitle', { table: value?.sourceTable ?? '' })}
        open={previewOpen}
        onCancel={() => setPreviewOpen(false)}
        footer={null}
        width={520}
      >
        {previewMutation.isPending ? (
          <div style={{ textAlign: 'center', padding: 24 }}>{t('common.loading')}</div>
        ) : previewMutation.data ? (
          <>
            <Table<DictionaryPreviewRow>
              size="small"
              columns={previewColumns}
              dataSource={previewMutation.data.rows.map((r, i) => ({ ...r, key: i }))}
              pagination={false}
              locale={{ emptyText: t('dictionary.source.previewEmpty') }}
            />
            <Typography.Text type="secondary" style={{ fontSize: 12, marginTop: 8, display: 'block' }}>
              {t('dictionary.source.previewSummary', {
                shown: previewMutation.data.rows.length,
                total: previewMutation.data.total,
              })}
            </Typography.Text>
          </>
        ) : null}
      </Modal>
    </>
  );
}
