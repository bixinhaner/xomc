import { useCallback, useMemo, useState } from 'react';
import { Alert, App, Button, List, Modal, Spin, Tag, Typography, Upload } from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import {
  CheckCircleOutlined,
  DownloadOutlined,
  InboxOutlined,
  UploadOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { deviceApi } from '@core/services/api/deviceApi';
import type {
  BatchImportDevice,
  BatchImportResponse,
  BatchImportRowError,
} from '@core/types/device';

const { Dragger } = Upload;
const { Text, Paragraph } = Typography;

export interface BatchImportModalProps {
  open: boolean;
  onClose: () => void;
  /**
   * 后端真实落库结束后回调（统计 + 错误明细），由父侧负责刷新列表与 toast。
   * 与旧版传 UploadFile[] 的 mock 行为不同——这里已不再需要原始文件对象。
   */
  onImport: (result: BatchImportResponse) => void | Promise<void>;
  onDownloadTemplate: () => void;
  t: (id: string, values?: Record<string, string | number>) => string;
  /** 用户当前选中的设备分组 ID；POST 时随 payload 一起带上，让后端把导入的
   * 设备一次性写入该分组。null/undefined 时设备保持"未分组"。 */
  selectedGroupId?: string | null;
}

// ── CSV header 与必填列定义 ───────────────────────────────────────────────────
//
// 表头别名 / 必填列 / 可识别列 全部从 deviceCsvSchema 取，跟导出 / 模板共用一份
// 元数据。同时支持「中文表头」（新模板 + 导出文件回灌）和「snake_case 表头」
// （老模板，向后兼容）。
import {
  HEADER_ALIAS,
  KNOWN_IMPORT_SNAKE,
  REQUIRED_IMPORT_SNAKE,
} from './deviceCsvSchema';

interface LocalParseError {
  row: number; // 1-based, 不含 header
  sn?: string;
  reason: string;
}

interface ParsedCsv {
  devices: BatchImportDevice[];
  localErrors: LocalParseError[];
}

// CSV 解析：剥 BOM → 跳注释 / 空行 → 切分 header + rows → 字段校验。
// 模板字段不含逗号，所以用 native split 即可（不引入 papaparse）。
function parseCsv(
  text: string,
  t: BatchImportModalProps['t']
): { ok: true; data: ParsedCsv } | { ok: false; reason: string } {
  // 1. 剥 UTF-8 BOM（U+FEFF）。用 charCodeAt 检查，避免在源码里出现裸 BOM
  // 触发 ESLint no-irregular-whitespace。
  const cleaned = text.charCodeAt(0) === 0xfeff ? text.slice(1) : text;
  // 2. 按 \r\n 或 \n 切行，过滤掉空行与 # 注释行
  const lines = cleaned
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l.length > 0 && !l.startsWith('#'));

  if (lines.length === 0) {
    return { ok: false, reason: t('device.batchImport.noDataRows') };
  }
  if (lines.length === 1) {
    // 只有 header 没数据
    return { ok: false, reason: t('device.batchImport.noDataRows') };
  }

  // header: 把每一列从「中文表头 / 老 snake_case 表头」归一化到 snake_case；
  // 未识别的列（如导出附带的展示列「连接状态/MAC地址/...」）保留原样，后续 KNOWN
  // 检查会把它们过滤掉。
  const rawHeader = lines[0].split(',').map((h) => h.trim());
  const header = rawHeader.map((h) => HEADER_ALIAS[h] ?? h);

  // 3. 校验 header 含必填列
  for (const required of REQUIRED_IMPORT_SNAKE) {
    if (!header.includes(required)) {
      return {
        ok: false,
        reason: t('device.batchImport.missingHeader', { field: required }),
      };
    }
  }

  // 4. 计算列下标
  const colIndex: Record<string, number> = {};
  header.forEach((col, idx) => {
    if (KNOWN_IMPORT_SNAKE.has(col)) {
      colIndex[col] = idx;
    }
  });

  // 5. 逐行解析与校验
  const devices: BatchImportDevice[] = [];
  const localErrors: LocalParseError[] = [];

  for (let i = 1; i < lines.length; i++) {
    const userRow = i; // CSV 第 1 数据行 = userRow 1
    const cells = lines[i].split(',').map((c) => c.trim());

    const get = (name: string): string | undefined => {
      const idx = colIndex[name];
      if (idx === undefined) return undefined;
      const v = cells[idx];
      return v === undefined || v === '' ? undefined : v;
    };

    const serialNumber = get('serial_number');

    // 必填校验：导入只需 SN（产品语义：按 SN 更新已有设备名称/备注 + 归入当前分组）。
    let validRow = true;
    for (const field of REQUIRED_IMPORT_SNAKE) {
      // 当前 REQUIRED_IMPORT_SNAKE = ['serial_number']；保持通用循环以便后续增列。
      if (!get(field)) {
        localErrors.push({
          row: userRow,
          sn: serialNumber,
          reason: t('device.batchImport.requiredMissing', { row: userRow, field }),
        });
        validRow = false;
      }
    }

    if (validRow && serialNumber) {
      devices.push({
        serial_number: serialNumber,
        device_name: get('device_name'),
        remark: get('remark'),
      });
    }
  }

  return { ok: true, data: { devices, localErrors } };
}

/**
 * 设备分组 - 批量导入弹窗。
 *
 * 流程（T-0202 真实化）：
 *   1. beforeUpload 时 FileReader 读取 CSV 文本，缓存解析结果
 *   2. 用户点确认 → 前端校验失败的行原样并入错误明细
 *   3. 通过前端校验的行打包 POST /devices/batch-import
 *   4. 合并前端 + 后端错误，展示统计 + 失败明细
 *
 * 不引入 papaparse；模板字段不含逗号，native split 足够。
 */
export default function BatchImportModal({
  open,
  onClose,
  onImport,
  onDownloadTemplate,
  t,
  selectedGroupId,
}: BatchImportModalProps) {
  // 注：之前直接用 `import { Modal } from 'antd'` 的 `Modal.error(...)` 静态方法
  // 弹反馈，但 antd v5 + React 19 下静态方法不会被 ConfigProvider 兼容层托管 →
  // 弹窗根本不渲染（参 commit 7f250967 T-0161）。用户看到的现象就是"选完文件
  // 没反应、导入按钮不亮、也没有任何报错"。统一改走 App.useApp().message。
  const { message } = App.useApp();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [parsed, setParsed] = useState<ParsedCsv | null>(null);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<BatchImportResponse | null>(null);

  const resetAll = useCallback(() => {
    setFileList([]);
    setParsed(null);
    setResult(null);
  }, []);

  const uploadProps: UploadProps = {
    name: 'file',
    multiple: false,
    accept: '.csv',
    fileList,
    beforeUpload: (file) => {
      // 后缀检查 case-insensitive：Windows Excel 导出 CSV 时常给到 `.CSV`，
      // 之前的 .endsWith('.csv') 会把它误判为非法格式直接 return false。
      if (!file.name.toLowerCase().endsWith('.csv')) {
        void message.error(t('device.fileFormatError'));
        return false;
      }
      const isLt10M = file.size / 1024 / 1024 < 10;
      if (!isLt10M) {
        void message.error(t('device.fileSizeError'));
        return false;
      }

      const reader = new FileReader();
      reader.onload = (e) => {
        const text = (e.target?.result as string) ?? '';
        const r = parseCsv(text, t);
        if (!r.ok) {
          void message.error(t('device.batchImport.parsingFailed', { reason: r.reason }));
          setFileList([]);
          setParsed(null);
          return;
        }
        setFileList([file]);
        setParsed(r.data);
        setResult(null);
      };
      reader.onerror = () => {
        void message.error(t('device.batchImport.parsingFailed', { reason: 'read file failed' }));
      };
      reader.readAsText(file, 'utf-8');
      return false;
    },
    onRemove: () => {
      resetAll();
    },
  };

  // 合并前端 + 后端错误（前端 row 是 CSV 行号，后端 row 是提交数组下标 + 1
  // —— 我们提交时只把"通过前端校验"的行打包，所以后端 row 不能直接复用为
  // CSV 行号。为了让用户体验一致，把 backend 失败行映射回它在原 devices[]
  // 里的 SN，然后通过 SN 找回 CSV 行号；找不到时退而展示后端原 row）。
  const mergedErrors = useMemo<BatchImportRowError[]>(() => {
    const front: BatchImportRowError[] = (parsed?.localErrors ?? []).map((e) => ({
      row: e.row,
      sn: e.sn,
      reason: e.reason,
    }));
    if (!result || !parsed) return front;
    // 把后端 row (1-based 索引到提交的 devices 数组) 映射回 CSV 行号
    // —— 我们在提交时已经把通过前端校验的行按顺序入了 parsed.devices；
    // 但 parsed.devices 没有携带"原 CSV 行号"，所以这里用 SN 兜底。
    // 若 SN 重复或匹配不到，就保留后端 row 原值（用户至少能看到 reason）。
    const back: BatchImportRowError[] = (result.errors ?? []).map((e) => ({
      row: e.row,
      sn: e.sn,
      reason: e.reason,
    }));
    return [...front, ...back];
  }, [parsed, result]);

  // 显示用：前 10 条 + 折叠余量
  const visibleErrors = useMemo(() => mergedErrors.slice(0, 10), [mergedErrors]);
  const hiddenErrorCount = Math.max(0, mergedErrors.length - visibleErrors.length);

  const handleImportConfirm = useCallback(async () => {
    if (fileList.length === 0 || !parsed) {
      void message.warning(t('device.selectFileFirst'));
      return;
    }

    // 边界：所有行都在前端被拦下，没有任何东西可以 POST
    if (parsed.devices.length === 0) {
      const stubResult: BatchImportResponse = {
        total: parsed.localErrors.length,
        succeeded: 0,
        failed: parsed.localErrors.length,
        errors: parsed.localErrors.map((e) => ({
          row: e.row,
          sn: e.sn,
          reason: e.reason,
        })),
      };
      setResult(stubResult);
      await onImport(stubResult);
      return;
    }

    setImporting(true);
    try {
      const resp = await deviceApi.batchImportDevices({
        devices: parsed.devices,
        // 仅在用户实际选中了某个分组时带上 group_id；undefined / null 时后端
        // 维持"导入即未分组"语义。
        ...(selectedGroupId ? { group_id: selectedGroupId } : {}),
      });
      // 把前端校验失败的行合并进 total / failed（让最终统计与 CSV 实际数据行一致）
      const finalResult: BatchImportResponse = {
        total: resp.total + parsed.localErrors.length,
        succeeded: resp.succeeded,
        failed: resp.failed + parsed.localErrors.length,
        errors: [
          ...parsed.localErrors.map((e) => ({
            row: e.row,
            sn: e.sn,
            reason: e.reason,
          })),
          ...(resp.errors ?? []),
        ],
      };
      setResult(finalResult);
      await onImport(finalResult);
    } catch (err) {
      void message.error(err instanceof Error ? err.message : String(err));
    } finally {
      setImporting(false);
    }
  }, [fileList, parsed, onImport, t, message, selectedGroupId]);

  const handleCancel = useCallback(() => {
    if (importing) return;
    resetAll();
    onClose();
  }, [importing, onClose, resetAll]);

  // 摘要面板：parsed 之后才显示
  const parseSummary = parsed && !result ? (
    <Alert
      style={{ marginBottom: 12 }}
      type={parsed.localErrors.length > 0 ? 'warning' : 'info'}
      showIcon
      icon={parsed.localErrors.length > 0 ? <WarningOutlined /> : undefined}
      message={
        <span>
          <Tag color="blue">{parsed.devices.length}</Tag>
          <Text type="secondary" style={{ fontSize: 13 }}>
            {t('device.batchImport.frontendBlocked', {
              count: parsed.localErrors.length,
            })}
          </Text>
        </span>
      }
    />
  ) : null;

  // 最终结果面板：result 已写
  const resultSummary = result ? (
    <Alert
      style={{ marginBottom: 12 }}
      type={result.failed === 0 ? 'success' : result.succeeded === 0 ? 'error' : 'warning'}
      showIcon
      icon={result.failed === 0 ? <CheckCircleOutlined /> : <WarningOutlined />}
      message={
        result.failed === 0
          ? t('device.batchImport.allSucceeded', { succeeded: result.succeeded })
          : result.succeeded === 0
            ? t('device.batchImport.allFailed', { failed: result.failed })
            : t('device.batchImport.partial', {
                succeeded: result.succeeded,
                failed: result.failed,
              })
      }
    />
  ) : null;

  // 错误明细
  const errorList = mergedErrors.length > 0 ? (
    <div style={{ marginBottom: 12 }}>
      <Paragraph style={{ marginBottom: 6, fontSize: 13 }} strong>
        {t('device.batchImport.errorListTitle')}
      </Paragraph>
      <List
        size="small"
        bordered
        dataSource={visibleErrors}
        renderItem={(item) => (
          <List.Item style={{ fontSize: 12 }}>
            <Text type="danger" style={{ marginRight: 8 }}>
              #{item.row}
            </Text>
            {item.sn ? <Tag style={{ marginRight: 8 }}>{item.sn}</Tag> : null}
            <Text>{item.reason}</Text>
          </List.Item>
        )}
      />
      {hiddenErrorCount > 0 ? (
        <Text type="secondary" style={{ fontSize: 12, marginTop: 4, display: 'block' }}>
          {t('device.batchImport.errorListMore', { n: hiddenErrorCount })}
        </Text>
      ) : null}
    </div>
  ) : null;

  return (
    <Modal
      title={t('common.batchImport')}
      open={open}
      onCancel={handleCancel}
      footer={null}
      width={600}
      maskClosable={!importing}
      closable={!importing}
      destroyOnHidden
    >
      <div style={{ marginBottom: 12 }}>
        <Button icon={<DownloadOutlined />} size="small" onClick={onDownloadTemplate}>
          {t('device.downloadImportTemplate')}
        </Button>
      </div>

      <Dragger {...uploadProps} style={{ marginBottom: 16 }}>
        <p className="ant-upload-drag-icon">
          <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
        </p>
        <p className="ant-upload-text">{t('common.upload')}</p>
        <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
          .csv
        </p>
      </Dragger>

      {parseSummary}
      {resultSummary}
      {errorList}

      {importing ? (
        <div style={{ marginBottom: 16, textAlign: 'center' }}>
          <Spin tip={t('common.loading')} />
        </div>
      ) : null}

      <Button
        type="primary"
        icon={<UploadOutlined />}
        loading={importing}
        onClick={() => void handleImportConfirm()}
        disabled={fileList.length === 0 || result !== null}
        block
      >
        {t('common.import')}
      </Button>

      {/* result !== null 后导入按钮置灰，提供「重选文件」入口让用户继续操作，
          避免回归 bug：导入完成后 modal 卡死，必须先关再开一遍。 */}
      {result !== null ? (
        <Button
          style={{ marginTop: 8 }}
          onClick={resetAll}
          block
        >
          {t('device.batchImport.reselectFile')}
        </Button>
      ) : null}
    </Modal>
  );
}
