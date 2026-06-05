/**
 * ParamModelPage — 参数模型主页面。
 *
 * 2026-05-28 用户决策(布局调整):
 *   1. "标准参数树"已拆为独立菜单页 product/standard-params,本页不再渲染
 *   2. drill-down 主从导航:
 *      - 默认显示参数模型清单(ModelsTab); 点击模型名 → 进入该模型的参数列表
 *      - 进入参数列表后,toolbar 左侧显示"返回"按钮 + 当前模型名;
 *        点击返回 → 清空 selectedModelName → 回到清单
 *   3. 顶部 toolbar 单行:
 *      - 列表态: [搜索 + 导入 XML 按钮]
 *      - 详情态: 由 MappingsTab 自带头部
 *
 * 三库 XML 导入重构(2026-06-04 D3/D5/D6):
 *   - 「导入 XML」改为弹框:必填「名称」文本框(目标文件名 = <名称>.xml)+ 选 XML 文件。
 *   - 单目录 + sidecar:上传写 param-mappings/<名称>.xml + <名称>.xml.custom 标记。
 *   - 双唯一性硬拒,无覆盖(无 force):
 *       · 提交前本地预检查清单是否已有同名 <名称>.xml → 内联报错"名称已存在,请改名"。
 *       · 后端 409(文件名或 XML 模型名已存在)→ 内联同款改名提示。
 *   - 后端上传端点内部自动 destructive 重载(删孤儿)+ 刷新缓存,前端无需单独调。
 */
import { useState } from 'react';
import { Card, Button, Space, message, Upload, Modal, Input, Form } from 'antd';
import type { UploadFile } from 'antd';
import { InboxOutlined, UploadOutlined } from '@ant-design/icons';
import {
  useParamModelList,
  useUploadParamModelXML,
} from '@core/hooks/api/useParamModels';
import type { AxiosError } from 'axios';
import ModelsTab from './ModelsTab';
import MappingsTab from './MappingsTab';
import SearchInput from '@/components/SearchInput';
import { useT } from '@/hooks/useT';

// 名称白名单:与后端 validateUploadFilename 的正则对齐(不含 .xml 扩展名部分)。
const NAME_PATTERN = /^[A-Za-z0-9_-]{1,64}$/;

export default function ParamModelPage() {
  const t = useT();
  const [selectedModelName, setSelectedModelName] = useState<string | undefined>();
  const [keyword, setKeyword] = useState('');
  const uploadMut = useUploadParamModelXML();
  // 上传前查重数据源:已存在的参数模型清单(按 loadedFrom basename 比对文件名)。
  const { data: modelListData } = useParamModelList();

  // 导入弹框状态
  const [importOpen, setImportOpen] = useState(false);
  const [importName, setImportName] = useState('');
  const [importFile, setImportFile] = useState<File | undefined>();
  const [nameError, setNameError] = useState<string | undefined>();

  const inDetail = Boolean(selectedModelName);

  // 文件名查重:与已存在模型的 loadedFrom basename 大小写不敏感比对(预检查)。
  const isDuplicateFile = (fileName: string): boolean => {
    const target = fileName.toLowerCase();
    return (modelListData?.items || []).some((m) => {
      const base = (m.loadedFrom || '').split('/').pop()?.toLowerCase() ?? '';
      return base === target;
    });
  };

  const resetImport = () => {
    setImportName('');
    setImportFile(undefined);
    setNameError(undefined);
  };

  const closeImport = () => {
    setImportOpen(false);
    resetImport();
  };

  // 名称变更:清掉上一次的内联错误,交给提交时再校验。
  const onNameChange = (v: string) => {
    setImportName(v);
    if (nameError) setNameError(undefined);
  };

  const submitImport = () => {
    const name = importName.trim();
    if (!name) {
      setNameError(t('product.paramModel.nameRequired'));
      return;
    }
    if (!NAME_PATTERN.test(name)) {
      setNameError(t('product.paramModel.nameInvalid'));
      return;
    }
    if (!importFile) {
      message.warning(t('product.paramModel.fileRequired'));
      return;
    }
    // 提交前本地查重:<name>.xml 已存在 → 内联报错,不发请求。
    if (isDuplicateFile(`${name}.xml`)) {
      setNameError(t('product.paramModel.nameExists'));
      return;
    }
    uploadMut
      .mutateAsync({ file: importFile, name })
      .then((r) => {
        message.success(t('product.paramModel.importSuccess', { file: r.filename }));
        closeImport();
      })
      .catch((e: unknown) => {
        const ax = e as AxiosError<{ message?: string }>;
        // 409 → 文件名或模型名已存在 → 内联改名提示(不覆盖)。
        if (ax.response?.status === 409) {
          setNameError(t('product.paramModel.nameExists'));
          return;
        }
        const msg =
          ax.response?.data?.message ?? (e instanceof Error ? e.message : String(e));
        message.error(msg);
      });
  };

  // antd Upload:仅用于选文件(beforeUpload 拦截自动上传),实际上传走 submitImport。
  const fileList: UploadFile[] = importFile
    ? [{ uid: '-1', name: importFile.name, status: 'done' }]
    : [];

  return (
    <div style={{ padding: 16 }}>
      {/* 顶部 toolbar:仅列表态。详情态由 MappingsTab 自带头部(返回 + 模型名 + 筛选/新增 同行) */}
      {!inDetail && (
        <Card size="small" style={{ marginBottom: 12 }}>
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <SearchInput
              placeholder={t('product.paramModel.searchPh')}
              allowClear
              onSearch={(v) => setKeyword(v.trim())}
              style={{ width: 320 }}
              enterButton
            />
            <Space>
              <Button icon={<InboxOutlined />} onClick={() => setImportOpen(true)}>
                {t('common.importXml')}
              </Button>
            </Space>
          </Space>
        </Card>
      )}

      {/* drill-down 主体:列表态 ↔ 详情态 二选一 */}
      {inDetail ? (
        <MappingsTab
          selectedName={selectedModelName}
          onBack={() => setSelectedModelName(undefined)}
        />
      ) : (
        <ModelsTab
          selectedName={selectedModelName}
          onSelect={setSelectedModelName}
          keyword={keyword}
        />
      )}

      {/* 导入 XML 弹框:必填名称 + 选文件;双唯一性硬拒,无覆盖。 */}
      <Modal
        title={t('common.importXml')}
        open={importOpen}
        onOk={submitImport}
        onCancel={closeImport}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        confirmLoading={uploadMut.isPending}
        destroyOnClose
      >
        <Form layout="vertical">
          <Form.Item
            label={t('product.paramModel.nameLabel')}
            required
            validateStatus={nameError ? 'error' : undefined}
            help={nameError ?? t('product.paramModel.nameHelp')}
          >
            <Input
              value={importName}
              onChange={(e) => onNameChange(e.target.value)}
              placeholder={t('product.paramModel.namePlaceholder')}
              maxLength={64}
              allowClear
            />
          </Form.Item>
          <Form.Item label={t('product.paramModel.fileLabel')} required>
            <Upload
              accept=".xml"
              maxCount={1}
              fileList={fileList}
              beforeUpload={(f) => {
                setImportFile(f as File);
                return false; // 阻止 antd 自动上传,文件由 submitImport 提交
              }}
              onRemove={() => setImportFile(undefined)}
            >
              <Button icon={<UploadOutlined />}>{t('product.paramModel.selectFile')}</Button>
            </Upload>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
