/**
 * 通用 Excel 导出工具（xlsx 0.18.x）。
 *
 * 用途：G6-Gap-10 panel 数据导出 + G7-Gap-4 adhoc 结果导出。
 *
 * 性能 / 健壮性（#15）：xlsx 体积较大，改为按需动态 import，避免进首包；
 * 且对动态 import 失败（依赖缺失 / chunk 加载失败 / 离线）做兜底——抛出
 * 可识别的 {@link ExcelExportUnavailableError}，由 UI 层捕获后用 i18n
 * 文案 `common.exportUnavailable`（导出暂不可用）友好提示，而不是整页崩溃
 * 或在打包阶段因解析不到模块而中断构建。
 */

export interface SheetInput {
  name: string;
  // 行对象数组（key = 列标题，value = 单元格内容）；null / undefined 自动空
  rows: Array<Record<string, string | number | null | undefined>>;
}

/**
 * 导出能力不可用错误。
 *
 * 当 xlsx 动态 import 失败（依赖缺失 / 网络分片加载失败等）时抛出。
 * UI 层据此判断并展示「导出暂不可用」（i18n key: `common.exportUnavailable`），
 * 而非把底层异常直接抛给用户。
 */
export class ExcelExportUnavailableError extends Error {
  /** 供 UI 层映射 i18n 文案的稳定标识 */
  readonly i18nKey = 'common.exportUnavailable';

  constructor(cause?: unknown) {
    super('Excel export module is unavailable');
    this.name = 'ExcelExportUnavailableError';
    // 保留底层原因，便于排查（不影响 UI 文案）
    if (cause !== undefined) {
      (this as { cause?: unknown }).cause = cause;
    }
  }
}

// xlsx 模块的最小契约（仅用到的成员），避免裸 any。
interface XlsxModule {
  utils: {
    book_new(): unknown;
    aoa_to_sheet(data: unknown[][]): unknown;
    json_to_sheet(rows: Array<Record<string, unknown>>): unknown;
    book_append_sheet(wb: unknown, ws: unknown, name: string): void;
  };
  writeFile(wb: unknown, filename: string): void;
}

/**
 * 按需加载 xlsx；加载失败统一包装为 ExcelExportUnavailableError。
 */
async function loadXlsx(): Promise<XlsxModule> {
  try {
    // 动态 import：失败（依赖缺失 / 分片加载失败）走 catch 兜底
    const mod = (await import('xlsx')) as unknown as XlsxModule;
    return mod;
  } catch (err) {
    throw new ExcelExportUnavailableError(err);
  }
}

/**
 * 导出多 sheet 的 .xlsx 文件。
 *
 * @param filename 用户下载文件名（不带扩展名；自动加 .xlsx）
 * @param sheets 多个 sheet 输入
 * @throws {ExcelExportUnavailableError} xlsx 不可用时（UI 层应捕获并提示「导出暂不可用」）
 */
export async function exportWorkbook(filename: string, sheets: SheetInput[]): Promise<void> {
  const XLSX = await loadXlsx();
  const wb = XLSX.utils.book_new();
  sheets.forEach((s) => {
    if (s.rows.length === 0) {
      // 空 sheet 仍创建（用户期望看到对应 sheet 即使是空的）
      const ws = XLSX.utils.aoa_to_sheet([['(no data)']]);
      XLSX.utils.book_append_sheet(wb, ws, sanitizeSheetName(s.name));
      return;
    }
    const ws = XLSX.utils.json_to_sheet(s.rows);
    XLSX.utils.book_append_sheet(wb, ws, sanitizeSheetName(s.name));
  });
  XLSX.writeFile(wb, `${filename}.xlsx`);
}

/**
 * 判断给定错误是否为「导出能力不可用」，供 UI 层统一兜底文案。
 */
export function isExcelExportUnavailable(err: unknown): err is ExcelExportUnavailableError {
  return err instanceof ExcelExportUnavailableError;
}

/**
 * Excel sheet 名禁止 \\ / ? * [ ] : 且长度 <= 31。
 */
function sanitizeSheetName(name: string): string {
  return name.replace(/[\\/?*[\]:]/g, '_').slice(0, 31) || 'Sheet';
}
