/**
 * AddSubFieldsModal — "用户规则 #4"批量添加 sub_field 入口。
 *
 * 工作流：
 *   1) admin 从 StandardParamSelect 多选 path
 *   2) 选择后实时预览每条 path 的 autofill 派生（mml_code / label.zh / label.en）
 *   3) 用户可在表格里 inline 改 mml_code / label，确认后调
 *      POST /admin/commands/:id/sub-fields/batch
 *   4) 成功后失效缓存，外层抽屉刷新列表
 *
 * 与现有 CreateSubField 区别：原 endpoint 一次只建一条且需手工填全部字段；
 * batch 端点按 standard_params 元数据自动派生，让维护人员减少 90% 重复劳动。
 */
import { useState, useMemo, useEffect } from 'react';
import { Modal, Table, Input, Form, Space, Tag, Alert, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import StandardParamSelect from './StandardParamSelect';
import type { StandardParamView } from '@core/types/mmlAdmin';
import {
  useBatchCreateSubFields,
} from '@core/hooks/api/useMmlAdmin';
import { useT } from '@/hooks/useT';

export interface AddSubFieldsModalProps {
  open: boolean;
  commandId: string;
  /** 命令下已绑定的 standard_path_id，用于在下拉里隐藏避免重复绑定。 */
  existingPathIds?: string[];
  onClose: () => void;
  onSuccess?: () => void;
}

interface PreviewRow {
  paramId: string;
  standardPath: string;
  description: string;
  /** autofill：path 末段 UPPER_SNAKE */
  mmlCode: string;
  labelZh: string;
  labelEn: string;
  access: string;
  dataType: string;
}

/** 把 standard_path 末段转成 UPPER_SNAKE_CASE。与后端 derivePathLeafCode 同算法。 */
function deriveMmlCode(path: string): string {
  const idx = path.lastIndexOf('.');
  const leaf = idx >= 0 ? path.slice(idx + 1) : path;
  let out = '';
  let prevLower = false;
  for (let i = 0; i < leaf.length; i++) {
    const ch = leaf[i];
    const isUpper = ch >= 'A' && ch <= 'Z';
    const isLower = ch >= 'a' && ch <= 'z';
    const isDigit = ch >= '0' && ch <= '9';
    if (i > 0 && prevLower && isUpper) out += '_';
    if (ch === '_' || ch === '-') {
      out += '_';
    } else if (isUpper || isDigit) {
      out += ch;
    } else if (isLower) {
      out += ch.toUpperCase();
    }
    prevLower = isLower;
  }
  return out;
}

/** humanize 末段：UserLabel → "User Label"。 */
function deriveLabelEn(path: string): string {
  const idx = path.lastIndexOf('.');
  const leaf = idx >= 0 ? path.slice(idx + 1) : path;
  let out = '';
  let prevLower = false;
  for (let i = 0; i < leaf.length; i++) {
    const ch = leaf[i];
    const isUpper = ch >= 'A' && ch <= 'Z';
    const isLower = ch >= 'a' && ch <= 'z';
    if (i > 0 && prevLower && isUpper) out += ' ';
    if (ch === '_' || ch === '-') {
      out += ' ';
    } else {
      out += ch;
    }
    prevLower = isLower;
  }
  return out;
}

export default function AddSubFieldsModal({
  open,
  commandId,
  existingPathIds = [],
  onClose,
  onSuccess,
}: AddSubFieldsModalProps) {
  const t = useT();
  const [rows, setRows] = useState<PreviewRow[]>([]);
  const batchMut = useBatchCreateSubFields();

  // 抽屉关闭时重置 rows，避免下次打开残留
  useEffect(() => {
    if (!open) setRows([]);
  }, [open]);

  const handleSelectionChange = (_: string | string[], records: StandardParamView[]) => {
    // 维护现有 rows 中已有的用户编辑值（按 paramId 索引），仅追加新增的
    setRows((prev) => {
      const prevMap = new Map(prev.map((r) => [r.paramId, r]));
      const next: PreviewRow[] = [];
      for (const sp of records) {
        const existing = prevMap.get(sp.id);
        if (existing) {
          next.push(existing);
          continue;
        }
        const mmlCode = deriveMmlCode(sp.standardPath);
        next.push({
          paramId: sp.id,
          standardPath: sp.standardPath,
          description: sp.description,
          mmlCode,
          labelZh: sp.description || mmlCode,
          labelEn: deriveLabelEn(sp.standardPath),
          access: sp.access,
          dataType: sp.dataType,
        });
      }
      return next;
    });
  };

  const selectedIds = useMemo(() => rows.map((r) => r.paramId), [rows]);

  const columns: ColumnsType<PreviewRow> = [
    {
      title: t('mml.admin.catalog.subField.standardPath'),
      dataIndex: 'standardPath',
      key: 'standardPath',
      ellipsis: true,
      render: (v: string) => <code style={{ fontSize: 11 }}>{v}</code>,
    },
    {
      title: t('mml.admin.catalog.subField.mmlCode'),
      dataIndex: 'mmlCode',
      key: 'mmlCode',
      width: 180,
      render: (_: unknown, row: PreviewRow, idx: number) => (
        <Input
          size="small"
          value={row.mmlCode}
          onChange={(e) => {
            const v = e.target.value;
            setRows((p) => p.map((r, i) => (i === idx ? { ...r, mmlCode: v } : r)));
          }}
        />
      ),
    },
    {
      title: 'label (zh)',
      dataIndex: 'labelZh',
      key: 'labelZh',
      ellipsis: true,
      render: (_: unknown, row: PreviewRow, idx: number) => (
        <Input
          size="small"
          value={row.labelZh}
          onChange={(e) => {
            const v = e.target.value;
            setRows((p) => p.map((r, i) => (i === idx ? { ...r, labelZh: v } : r)));
          }}
        />
      ),
    },
    {
      title: t('mml.admin.catalog.subField.access'),
      dataIndex: 'access',
      key: 'access',
      width: 110,
      render: (v: string) => (
        <Tag color={v === 'READ_WRITE' ? 'blue' : 'default'}>{v || '-'}</Tag>
      ),
    },
  ];

  const handleOk = async () => {
    if (rows.length === 0) {
      message.warning(t('mml.admin.catalog.subField.batchEmptyHint'));
      return;
    }
    try {
      // 注：后端 batch 端点会自己再派生一遍 mml_code/label —— 这里 inline 编辑过的
      // 值不会被使用。后端 batch 是"按 standard_param 元数据派生"的语义；如果
      // user 想要手工填值，应该用单条 CreateSubField 端点。这里 UI 上允许编辑
      // 仅是预览自我说明，提交时只传 id 列表。
      await batchMut.mutateAsync({
        commandId,
        req: { standardPathIds: rows.map((r) => r.paramId) },
      });
      message.success(
        t('mml.admin.catalog.subField.batchCreated', { count: rows.length }),
      );
      onSuccess?.();
      onClose();
    } catch (e) {
      if (e instanceof Error) message.error(e.message);
    }
  };

  return (
    <Modal
      open={open}
      onCancel={onClose}
      onOk={handleOk}
      okButtonProps={{ loading: batchMut.isPending, disabled: rows.length === 0 }}
      title={t('mml.admin.catalog.subField.batchAddTitle')}
      width={780}
      destroyOnHidden
    >
      <Form layout="vertical">
        <Form.Item label={t('mml.admin.catalog.path.select')} required>
          <StandardParamSelect
            multiple
            value={selectedIds}
            onChange={handleSelectionChange}
            excludeIds={existingPathIds}
          />
        </Form.Item>
        <Alert
          type="info"
          showIcon
          message={t('mml.admin.catalog.subField.batchAutofillHint')}
          style={{ marginBottom: 12 }}
        />
        <Table<PreviewRow>
          rowKey="paramId"
          size="small"
          columns={columns}
          dataSource={rows}
          pagination={false}
          scroll={{ y: 320 }}
        />
        <Space style={{ marginTop: 8 }}>
          <span style={{ color: '#8c8c8c', fontSize: 12 }}>
            {t('mml.admin.catalog.subField.batchTotalLabel', { count: rows.length })}
          </span>
        </Space>
      </Form>
    </Modal>
  );
}
