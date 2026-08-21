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
 * 导入 XML 调整(2026-06-05 取消手填名称):
 *   - 「导入 XML」弹框只选 XML 文件;唯一名称取自 XML <parameterModel paramModel="...">
 *     属性,文件将保存为 <paramModel>.xml(选文件后即时预览)。
 *   - 单目录 + sidecar:上传写 param-mappings/<paramModel>.xml(+ .custom 标记,仅新建)。
 *   - 重复允许覆盖(二次确认):本地预检(清单 name / 文件名比对)或后端 409
 *     (data.overwritable=true)→ Modal.confirm「已存在,确认覆盖?」→ 确认后带
 *     force=true 重试,后端覆盖归属文件并自动备份旧文件 .bak.<ts>。
 *   - 后端上传端点内部自动 destructive 重载(删孤儿)+ 刷新缓存,前端无需单独调。
 */
import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Card, Button, Space, message, Upload, Modal, Form, Typography } from 'antd';
import type { UploadFile } from 'antd';
import { InboxOutlined, UploadOutlined } from '@ant-design/icons';
import {
  useParamModelList,
  useUploadParamModelXML,
} from '@core/hooks/api/useParamModels';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';
import type { AxiosError } from 'axios';
import {
  filterDeviceScopedItemsByLicense,
  isDeviceStandardValueVisibleByLicense,
} from '@core/utils/licenseFeatures';
import { extractXmlRootAttr } from '@core/utils/xmlRootAttr';
import ModelsTab from './ModelsTab';
import MappingsTab from './MappingsTab';
import SearchInput from '@/components/SearchInput';
import { useT } from '@/hooks/useT';

export default function ParamModelPage() {
  const t = useT();
  // drill-down 状态走 URL query(?name=<模型名>),刷新/分享链接保留二级页(参照 kpi-library)。
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedModelName = searchParams.get('name') || undefined;
  const selectModel = (name?: string) => {
    const params = new URLSearchParams(searchParams);
    if (name) {
      params.set('name', name);
    } else {
      params.delete('name');
    }
    setSearchParams(params, { replace: false });
  };
  const [keyword, setKeyword] = useState('');
  const uploadMut = useUploadParamModelXML();
  // 上传前查重数据源:已存在的参数模型清单(按 name / loadedFrom basename 比对)。
  const { data: modelListData } = useParamModelList();
  const { data: systemLicense, isLoading: systemLicenseLoading } = useSystemLicense();
  const visibleModels = useMemo(
    () => filterDeviceScopedItemsByLicense(modelListData?.items || [], systemLicense, systemLicenseLoading),
    [modelListData, systemLicense, systemLicenseLoading],
  );
  const visibleModelNames = useMemo(() => visibleModels.map((model) => model.name), [visibleModels]);

  // 导入弹框状态
  const [importOpen, setImportOpen] = useState(false);
  const [importFile, setImportFile] = useState<File | undefined>();
  // 选文件后从 XML paramModel 属性提取的名称(将保存为 <paramModel>.xml);null = 未提取到
  const [derivedName, setDerivedName] = useState<string | null>(null);
  const [importError, setImportError] = useState<string | undefined>();

  const selectedModelVisible = !selectedModelName
    || isDeviceStandardValueVisibleByLicense(selectedModelName, systemLicense, systemLicenseLoading);
  const inDetail = Boolean(selectedModelName && selectedModelVisible);

  useEffect(() => {
    if (systemLicenseLoading || !selectedModelName) return;
    if (isDeviceStandardValueVisibleByLicense(selectedModelName, systemLicense, false)) return;
    const params = new URLSearchParams(searchParams);
    params.delete('name');
    setSearchParams(params, { replace: true });
  }, [searchParams, selectedModelName, setSearchParams, systemLicense, systemLicenseLoading]);

  useEffect(() => {
    if (!derivedName || systemLicenseLoading) return;
    if (!isDeviceStandardValueVisibleByLicense(derivedName, systemLicense, false)) {
      setImportError(t('systemLicense.deviceStandardImportDenied', { standard: derivedName }));
    }
  }, [derivedName, systemLicense, systemLicenseLoading, t]);

  // 查重:与已存在模型的 name / loadedFrom basename 大小写不敏感比对(预检查)。
  const isDuplicate = (name: string): boolean => {
    const lower = name.toLowerCase();
    const target = `${lower}.xml`;
    return (modelListData?.items || []).some((m) => {
      const base = (m.loadedFrom || '').split('/').pop()?.toLowerCase() ?? '';
      return base === target || m.name.toLowerCase() === lower;
    });
  };

  const resetImport = () => {
    setImportFile(undefined);
    setDerivedName(null);
    setImportError(undefined);
  };

  const closeImport = () => {
    setImportOpen(false);
    resetImport();
  };

  // 实际上传(force = 二次确认后的覆盖);成功后关弹窗,失败统一提示。
  const doUpload = (file: File, force: boolean) => {
    uploadMut
      .mutateAsync({ file, force })
      .then((r) => {
        message.success(
          r.overwritten
            ? t('product.upload.overwriteSuccess', { file: r.filename })
            : t('product.paramModel.importSuccess', { file: r.filename }),
        );
        closeImport();
      })
      .catch((e: unknown) => {
        const ax = e as AxiosError<{ msg?: string; message?: string; data?: { overwritable?: boolean } }>;
        const body = ax.response?.data;
        const msg =
          body?.msg ?? body?.message ?? (e instanceof Error ? e.message : String(e));
        // 409 + overwritable(本地预检漏判的重复)→ 弹确认覆盖
        if (!force && ax.response?.status === 409 && body?.data?.overwritable) {
          confirmOverwrite(derivedName ?? '', () => doUpload(file, true));
          return;
        }
        message.error(msg);
      });
  };

  // 覆盖二次确认弹框 —— 确认后才带 force=true 重试。
  const confirmOverwrite = (name: string, onOk: () => void) => {
    Modal.confirm({
      title: t('product.upload.overwriteConfirmTitle'),
      content: t('product.upload.overwriteConfirmContent', { name }),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk,
    });
  };

  const submitImport = () => {
    if (!importFile) {
      message.warning(t('product.paramModel.fileRequired'));
      return;
    }
    // paramModel 属性是名称唯一来源 —— 读不到直接拒绝(后端也会 400)。
    if (!derivedName) {
      setImportError(t('product.upload.missingAttr', { attr: 'paramModel' }));
      return;
    }
    if (systemLicenseLoading) {
      setImportError(t('common.loading'));
      return;
    }
    if (!isDeviceStandardValueVisibleByLicense(derivedName, systemLicense, false)) {
      setImportError(t('systemLicense.deviceStandardImportDenied', { standard: derivedName }));
      return;
    }
    // 提交前本地查重:模型已存在 → 弹覆盖确认(后端 409 仍是兜底真值源)。
    if (isDuplicate(derivedName)) {
      confirmOverwrite(derivedName, () => doUpload(importFile, true));
      return;
    }
    doUpload(importFile, false);
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
          onBack={() => selectModel(undefined)}
        />
      ) : (
        <ModelsTab
          selectedName={selectedModelName}
          onSelect={selectModel}
          keyword={keyword}
          visibleModelNames={visibleModelNames}
        />
      )}

      {/* 导入 XML 弹框:仅选文件(名称取自 XML paramModel 属性);双唯一性硬拒,无覆盖。 */}
      <Modal
        title={t('common.importXml')}
        open={importOpen}
        onOk={submitImport}
        onCancel={closeImport}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        confirmLoading={uploadMut.isPending}
        destroyOnHidden
      >
        <Form layout="vertical">
          <Form.Item
            label={t('product.paramModel.fileLabel')}
            required
            validateStatus={importError ? 'error' : undefined}
            help={importError ?? t('product.paramModel.nameAutoHint')}
          >
            <Upload
              accept=".xml"
              maxCount={1}
              fileList={fileList}
              beforeUpload={(f) => {
                setImportFile(f as File);
                setImportError(undefined);
                // 选文件即抽 paramModel 属性,预览"将保存为 <paramModel>.xml"
                void extractXmlRootAttr(f as File, 'paramModel').then((n) => setDerivedName(n));
                return false; // 阻止 antd 自动上传,文件由 submitImport 提交
              }}
              onRemove={() => {
                setImportFile(undefined);
                setDerivedName(null);
                setImportError(undefined);
              }}
            >
              <Button icon={<UploadOutlined />}>{t('product.paramModel.selectFile')}</Button>
            </Upload>
            {derivedName && (
              <Typography.Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                {t('product.upload.savedAs', { name: derivedName })}
              </Typography.Text>
            )}
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
