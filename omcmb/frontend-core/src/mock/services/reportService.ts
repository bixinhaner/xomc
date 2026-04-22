import type { PageRequest, PageResponse } from '../../types/pagination';
import type { ReportDefinition, ReportRecord, ReportType, ReportPeriod } from '../data/reports';
import { mockReportDefinitions, mockReportRecords, mockReportSampleData } from '../data/reports';
import { delay, paginate, generateId } from '../utils';

let reportDefs = [...mockReportDefinitions];
const reportRecords = [...mockReportRecords];

export const reportService = {
  async getDefinitions(
    params: { reportType?: ReportType; status?: string } & PageRequest
  ): Promise<PageResponse<ReportDefinition>> {
    await delay(100, 200);
    let filtered = [...reportDefs];
    if (params.reportType) filtered = filtered.filter((r) => r.reportType === params.reportType);
    if (params.status) filtered = filtered.filter((r) => r.status === params.status);
    return paginate(filtered, params.page, params.pageSize);
  },

  async getDefinitionById(id: string): Promise<ReportDefinition | null> {
    await delay(80, 150);
    return reportDefs.find((r) => r.id === id) ?? null;
  },

  async createDefinition(data: Omit<ReportDefinition, 'id' | 'createTime'>): Promise<ReportDefinition> {
    await delay(200, 400);
    const newItem: ReportDefinition = {
      ...data,
      id: generateId('rdef'),
      createTime: new Date().toISOString(),
    };
    reportDefs.push(newItem);
    return newItem;
  },

  async updateDefinition(id: string, data: Partial<ReportDefinition>): Promise<ReportDefinition> {
    await delay(150, 300);
    const idx = reportDefs.findIndex((r) => r.id === id);
    if (idx === -1) throw new Error(`Report definition ${id} not found`);
    reportDefs[idx] = { ...reportDefs[idx], ...data };
    return reportDefs[idx];
  },

  async deleteDefinitions(ids: string[]): Promise<void> {
    await delay(150, 300);
    reportDefs = reportDefs.filter((r) => !ids.includes(r.id));
  },

  async getRecords(
    params: { reportDefinitionId?: string; format?: string } & PageRequest
  ): Promise<PageResponse<ReportRecord>> {
    await delay(100, 200);
    let filtered = [...reportRecords];
    if (params.reportDefinitionId) filtered = filtered.filter((r) => r.reportDefinitionId === params.reportDefinitionId);
    if (params.format) filtered = filtered.filter((r) => r.format === params.format);
    return paginate(filtered, params.page, params.pageSize);
  },

  async generateReport(
    definitionId: string,
    period?: string,
    reportType?: ReportPeriod
  ): Promise<ReportRecord> {
    await delay(1000, 3000);
    void reportType;
    const def = reportDefs.find((r) => r.id === definitionId);
    const newRecord: ReportRecord = {
      id: generateId('rrec'),
      reportDefinitionId: definitionId,
      reportName: `${def?.reportName ?? '报表'}-${period ?? new Date().toISOString().slice(0, 10)}`,
      period: period ?? new Date().toISOString().slice(0, 10),
      generateTime: new Date().toISOString(),
      fileSize: 1024 * 1024 * (Math.random() * 5 + 1) | 0,
      downloadUrl: `/reports/generated/${generateId('report')}.pdf`,
      format: 'pdf',
      status: 'ready',
    };
    reportRecords.push(newRecord);
    if (def) {
      const defIdx = reportDefs.findIndex((r) => r.id === definitionId);
      if (defIdx !== -1) {
        reportDefs[defIdx] = { ...reportDefs[defIdx], lastGenTime: newRecord.generateTime };
      }
    }
    return newRecord;
  },

  async downloadRecord(id: string): Promise<{ url: string; fileName: string }> {
    await delay(200, 500);
    const record = reportRecords.find((r) => r.id === id);
    if (!record) throw new Error(`Report record ${id} not found`);
    return { url: record.downloadUrl, fileName: `${record.reportName}.${record.format}` };
  },

  async getSampleData(): Promise<typeof mockReportSampleData> {
    await delay(80, 150);
    return mockReportSampleData;
  },
};
