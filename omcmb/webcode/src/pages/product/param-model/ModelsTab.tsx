import { useMemo, useState } from 'react';
import { Card, Table, Tag, Button, Space, Modal, Form, Input, Switch, message, Popconfirm, Tooltip } from 'antd';
import { EditOutlined, DeleteOutlined, DownloadOutlined } from '@ant-design/icons';
import {
  useParamModelList,
  useUpdateParamModel,
  useDeleteParamModel,
  useDownloadParamModelXML,
} from '@core/hooks/api/useParamModels';
import type { ParamModel, UpdateParamModelInput } from '@core/types/paramModel';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import { useT } from '@/hooks/useT';
import {
  PRODUCT_TABLE_DEFAULT_PAGE_SIZE,
  PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
} from '../pagination';

interface Props {
  selectedName?: string;
  onSelect: (name: string) => void;
  /** 2026-05-28: 由 param-model index 顶部统一 toolbar 提供的关键字过滤
   *  (取消 Tabs 后,搜索框上提到容器外,client-side 过滤 name / description)。 */
  keyword?: string;
  /** License 过滤后的模型名集合；未传表示不过滤。 */
  visibleModelNames?: readonly string[];
}

export default function ModelsTab({ selectedName, onSelect, keyword, visibleModelNames }: Props) {
  const t = useT();
  const { data, isLoading } = useParamModelList();
  const updateMut = useUpdateParamModel();
  const deleteMut = useDeleteParamModel();
  const downloadMut = useDownloadParamModelXML();

  const [editing, setEditing] = useState<ParamModel | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PRODUCT_TABLE_DEFAULT_PAGE_SIZE);
  const [form] = Form.useForm<UpdateParamModelInput>();

  // 关键字变化时回到第一页(避免过滤后停留在空页)——渲染期重置,避免 set-state-in-effect。
  const [prevKeyword, setPrevKeyword] = useState(keyword);
  if (keyword !== prevKeyword) {
    setPrevKeyword(keyword);
    setPage(1);
  }

  const visibleNameSet = useMemo(() => (
    visibleModelNames
      ? new Set(visibleModelNames.map((name) => name.trim().toUpperCase()))
      : undefined
  ), [visibleModelNames]);

  const items = useMemo(() => {
    // 2026-05-29 用户决策:前端隐藏"无加载源"(source=unknown)的孤儿模型。
    // 后端 ListParamModels SQL 已加 WHERE 前缀过滤 + migration 218 一次性物理清理,
    // 本 .filter 是双保险,防止 stale cache / 老版本 API 漏出 unknown 行。
    const raw = (data?.items || [])
      .filter((m) => (m.source ?? 'unknown') !== 'unknown')
      .filter((m) => !visibleNameSet || visibleNameSet.has(m.name.trim().toUpperCase()));
    const k = keyword?.trim().toLowerCase();
    if (!k) return raw;
    return raw.filter((m) =>
      (m.name + ' ' + (m.description || '')).toLowerCase().includes(k),
    );
  }, [data, keyword, visibleNameSet]);

  const columns = [
    {
      title: t('common.name'),
      dataIndex: 'name',
      width: 240,
      render: (v: string, row: ParamModel) => (
        <Button
          type="link"
          size="small"
          onClick={() => onSelect(row.name)}
          style={{ padding: 0, fontWeight: row.name === selectedName ? 600 : 400 }}
        >
          {v}
        </Button>
      ),
    },
    // 2026-06-03 用户决策:去掉"来源(builtin/custom)"列,保留"加载源(loaded_from)"列。
    { title: t('common.loadedFrom'), dataIndex: 'loadedFrom', width: 260, ellipsis: true },
    { title: t('product.paramModel.models.colTotalEntries'), dataIndex: 'totalEntries', width: 90 },
    { title: t('product.paramModel.models.colTotalObjects'), dataIndex: 'totalObjects', width: 90 },
    { title: t('product.paramModel.models.colTotalParams'), dataIndex: 'totalParams', width: 90 },
    {
      title: t('common.activate'),
      dataIndex: 'isActive',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="success">{t('common.active')}</Tag> : <Tag>{t('common.inactive')}</Tag>),
    },
    { title: t('common.description'), dataIndex: 'description', ellipsis: true },
    {
      title: t('common.action'),
      width: 190,
      render: (_: unknown, row: ParamModel) => (
        <Space>
          {/* 2026-06-05:下载 XML 原文件(builtin / custom 均可) */}
          <Tooltip title={t('product.upload.downloadXml')}>
            <Button
              size="small"
              icon={<DownloadOutlined />}
              disabled={!row.loadedFrom}
              onClick={() =>
                downloadMut
                  .mutateAsync({ loadedFrom: row.loadedFrom })
                  .catch((e) => message.error((e as Error).message))
              }
            />
          </Tooltip>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditing(row);
              form.setFieldsValue({ description: row.description, isActive: row.isActive });
            }}
          />
          {/* 2026-06-04 用户决策:内置(builtin)不可删 → 仅 custom 可删,内置置灰 + Tooltip。 */}
          {row.deletable ? (
          <Popconfirm
            title={t('product.paramModel.models.confirmDeleteCustom', { name: row.name })}
            description={
              <div style={{ maxWidth: 320 }}>
                · {t('product.paramModel.models.deleteBullet1Pre')} <code>.deleted.&lt;ts&gt;</code> {t('product.paramModel.models.deleteBullet1Post')}
                <br />· {t('product.paramModel.cascadeHint')}
                <br />· {t('product.paramModel.models.deleteBullet2')}
              </div>
            }
            okButtonProps={{ danger: true }}
            okText={t('product.paramModel.delConfirm')}
            onConfirm={() =>
              deleteMut
                .mutateAsync(row.name)
                .then(() => message.success(t('common.deleted')))
                .catch((e) => message.error((e as Error).message))
            }
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
          ) : (
            <Tooltip title={t('common.builtinNoDelete')}>
              <Button size="small" danger icon={<DeleteOutlined />} disabled />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  const handleSave = async () => {
    if (!editing) return;
    try {
      const v = await form.validateFields();
      await updateMut.mutateAsync({ name: editing.name, input: v });
      message.success(t('common.saved'));
      setEditing(null);
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  return (
    <Card size="small">
      <Table<ParamModel>
        rowKey="id"
        loading={isLoading}
        columns={[makeSeqColumn<ParamModel>({ title: t('table.rowNumber'), dataSource: items }), ...columns]}
        dataSource={items}
        size="small"
        pagination={{
          current: page,
          pageSize,
          total: items.length,
          showSizeChanger: true,
          pageSizeOptions: PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
          showTotal: (n) => t('common.totalCount', { count: n }),
          onChange: (p, ps) => {
            setPage(p);
            if (ps !== pageSize) setPageSize(ps);
          },
        }}
      />
      <Modal
        title={t('product.paramModel.models.editTitle', { name: editing?.name ?? '' })}
        open={Boolean(editing)}
        onOk={() => void handleSave()}
        onCancel={() => setEditing(null)}
        confirmLoading={updateMut.isPending}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="isActive" label={t('common.active')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
