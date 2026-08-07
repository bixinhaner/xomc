// #318-321 资源保留与上传背压配置（系统配置页 —— 一个 tab，4 张分类卡片）。
//
// 后端 sys_configs 分类（seed migrations/seed/000004，各模块热加载）：
//   acs.backpressure     #318 PM 上传背压 watchdog（ACS 每采样周期轮询刷新）
//   minio.retention      #319 原始件 ILM 保留天数（app RegisterSavedHook 重应用 lifecycle）
//   stationlog.retention #320+#798 基站日志按时间保留 + 全局/每设备文件数配额（worker TTL 缓存）
//   raw_archive          #836 PM/MR 新文件入库后一次性压缩开关（worker TTL 缓存）
//
// 接入方式与 PmRetentionSection 一致：复用 useSysConfigsByCategory + useBatchUpdateSysConfigs，
// 每张卡片自管 form + 保存（一次保存 = 该分类一次 batch upsert）。

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Card, Form, InputNumber, Switch, message, Spin } from 'antd';
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem';
import { useT } from '@/hooks/useT';

type FieldType = 'int' | 'float' | 'bool';

interface FieldSpec {
  key: string;
  type: FieldType;
  min?: number;
  max?: number;
  step?: number;
}

interface CardSpec {
  category: string;
  titleKey: string;
  descKey: string;
  fields: FieldSpec[];
}

// 字段定义与后端 sys_configs 键 + value_type 严格对齐（seed 000004）。
const CARDS: CardSpec[] = [
  {
    category: 'acs.backpressure',
    titleKey: 'retentionBp.bp.title',
    descKey: 'retentionBp.bp.desc',
    fields: [
      { key: 'enabled', type: 'bool' },
      { key: 'disk_high_pct', type: 'int', min: 1, max: 100 },
      { key: 'disk_low_pct', type: 'int', min: 0, max: 100 },
      { key: 'check_interval_sec', type: 'int', min: 1, max: 3600 },
    ],
  },
  {
    category: 'minio.retention',
    titleKey: 'retentionBp.ilm.title',
    descKey: 'retentionBp.ilm.desc',
    fields: [{ key: 'raw_object_days', type: 'int', min: 1, max: 3650 }],
  },
  {
    category: 'stationlog.retention',
    titleKey: 'retentionBp.stationlog.title',
    descKey: 'retentionBp.stationlog.desc',
    fields: [
      { key: 'max_retention_days', type: 'int', min: 1, max: 3650 },
      { key: 'max_file_count', type: 'int', min: 0, max: 100000 },
      { key: 'max_file_count_per_device', type: 'int', min: 0, max: 100000 },
      { key: 'cleanup_interval_minutes', type: 'int', min: 10, max: 1440 },
    ],
  },
  {
    category: 'raw_archive',
    titleKey: 'retentionBp.archive.title',
    descKey: 'retentionBp.archive.desc',
    fields: [{ key: 'compress_after_ingest', type: 'bool' }],
  },
];

function decodeValue(raw: string, type: FieldType): number | boolean {
  if (type === 'bool') {
    return raw === 'true' || raw === '1';
  }
  const n = type === 'int' ? parseInt(raw, 10) : parseFloat(raw);
  return Number.isFinite(n) ? n : 0;
}

function encodeValue(v: unknown, type: FieldType): string {
  if (type === 'bool') {
    return v ? 'true' : 'false';
  }
  return String(v ?? '');
}

function CategoryConfigCard({ spec }: { spec: CardSpec }) {
  const t = useT();
  const [form] = Form.useForm<Record<string, number | boolean>>();
  const [submitting, setSubmitting] = useState(false);

  const {
    data: configs,
    isLoading,
    isFetching,
    isError,
    isSuccess,
    refetch,
  } = useSysConfigsByCategory(spec.category);
  const { mutateAsync: batchUpdate } = useBatchUpdateSysConfigs();
  const canEdit = isSuccess && !isFetching;
  const canEditRef = useRef(canEdit);
  useLayoutEffect(() => {
    canEditRef.current = canEdit;
  }, [canEdit]);

  // 把后端 sys_configs 行映射为 { key: typed value }；缺失键给安全默认。
  const initialValues = useMemo<Record<string, number | boolean>>(() => {
    const out: Record<string, number | boolean> = {};
    for (const f of spec.fields) {
      out[f.key] = f.type === 'bool' ? false : 0;
    }
    if (configs) {
      for (const cfg of configs) {
        const f = spec.fields.find((x) => x.key === cfg.key);
        if (f) {
          out[cfg.key] = decodeValue(cfg.value, f.type);
        }
      }
    }
    return out;
  }, [configs, spec.fields]);

  useEffect(() => {
    if (!isSuccess) return;
    form.setFields(
      spec.fields.map((field) => ({
        name: field.key,
        value: initialValues[field.key],
        touched: false,
      })),
    );
  }, [form, initialValues, isSuccess, spec.fields]);

  const handleSave = async () => {
    if (!canEditRef.current) {
      message.error(t('empty.loadFailed'));
      return;
    }
    try {
      const values = await form.validateFields();
      if (!canEditRef.current) {
        message.error(t('empty.loadFailed'));
        return;
      }
      const changedFields = spec.fields.filter((field) => form.isFieldTouched(field.key));
      if (changedFields.length === 0) {
        message.warning(t('sysconfig.warn.noSaveable'));
        return;
      }
      setSubmitting(true);
      await batchUpdate({
        category: spec.category,
        items: changedFields.map((f) => ({
          key: f.key,
          value: encodeValue(values[f.key], f.type),
          value_type: f.type,
        })),
      });
      message.success(t('retentionBp.save.success'));
    } catch {
      // validateFields 失败或网络错误：交给 antd 自身 UI 提示
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card
      size="small"
      title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t(spec.titleKey)}</span>}
      style={{ marginBottom: 16 }}
      extra={
        <Button size="small" type="primary" onClick={handleSave} loading={submitting} disabled={!canEdit}>
          {t('retentionBp.save')}
        </Button>
      }
    >
      <div style={{ color: '#888', fontSize: 12, marginBottom: 12 }}>{t(spec.descKey)}</div>
      {isError && (
        <Alert
          type="error"
          showIcon
          title={t('empty.loadFailed')}
          description={t('empty.loadFailedDesc')}
          action={<Button size="small" onClick={() => void refetch()}>{t('common.retry')}</Button>}
          style={{ marginBottom: 12 }}
        />
      )}
      <Spin spinning={isLoading || isFetching}>
        <Form form={form} layout="vertical" size="small">
          {spec.fields.map((f) => (
            <Form.Item
              key={f.key}
              name={f.key}
              label={t(`retentionBp.field.${f.key}`)}
              valuePropName={f.type === 'bool' ? 'checked' : 'value'}
              rules={f.type === 'bool' ? undefined : [
                {
                  type: 'number',
                  min: f.min,
                  max: f.max,
                  message: t('retentionBp.validate.range', {
                    field: t(`retentionBp.field.${f.key}`),
                    min: f.min ?? 0,
                    max: f.max ?? 0,
                  }),
                },
              ]}
              style={{ marginBottom: 12, maxWidth: 360 }}
            >
              {f.type === 'bool' ? (
                <Switch />
              ) : (
                <InputNumber min={f.min} max={f.max} step={f.step ?? 1} style={{ width: 200 }} />
              )}
            </Form.Item>
          ))}
        </Form>
      </Spin>
    </Card>
  );
}

export default function RetentionBackpressureSection() {
  return (
    <>
      {CARDS.map((spec) => (
        <CategoryConfigCard key={spec.category} spec={spec} />
      ))}
    </>
  );
}
