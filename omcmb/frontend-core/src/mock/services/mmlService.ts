import type { MMLCommand, MMLScript, MMLTask, MMLResult, MMLCustomCommand, MMLTaskCreateInput, DeviceTaskResultItem, MMLTaskResultsStats } from '../../types/mml';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { mockMMLCommands, mockMMLScripts, mockMMLTasks } from '../data/mml';
import { delay, paginate, generateId } from '../utils';

let scripts = [...mockMMLScripts];
let tasks = [...mockMMLTasks];

function generateMMLOutput(commandCode: string, deviceSn: string): MMLResult {
  const outputs: Record<string, string> = {
    'DSP VERSION': `VERSION: V100R011C10SPC200\nDEVICE: ${deviceSn}\nBUILD: 20240315\nSTATUS: Normal`,
    'DSP BOARDSTATUS': `SRN=0 SN=0 TYPE=BBU STATUS=NORMAL\nSRN=0 SN=1 TYPE=UMPT STATUS=NORMAL\nSRN=0 SN=2 TYPE=UBBP STATUS=NORMAL`,
    'LST CELL': `CELLID=0 STATUS=ACTIVE ADMINSTATE=UNLOCKED PCI=128 TXPOWER=43\nCELLID=1 STATUS=ACTIVE ADMINSTATE=UNLOCKED PCI=129 TXPOWER=43`,
    'ACT CELL': `RETCODE=0\nCELLID=0 STATUS CHANGE: INACTIVE -> ACTIVE`,
    'DEA CELL': `RETCODE=0\nCELLID=0 STATUS CHANGE: ACTIVE -> INACTIVE`,
    'MOD CELL': `RETCODE=0\nCELL CONFIGURATION UPDATED`,
    'RST CELL': `RETCODE=0\nCELL RESET SUCCESSFULLY`,
    'LST NCELL': `LOCALCELLID=0 NCELLID=128 PRIORITY=1\nLOCALCELLID=0 NCELLID=129 PRIORITY=2`,
    'ADD NCELL': `RETCODE=0\nNEIGHBOR CELL RELATION ADDED`,
    'DEL NCELL': `RETCODE=0\nNEIGHBOR CELL RELATION DELETED`,
    'LST BTSSTATE': `BTS_STATUS=NORMAL\nRUNTIME=30d 12h 45m\nLAST_RESTART=2024-06-01T08:00:00`,
    'DSP SYSRESOURCE': `CPU_USAGE=45%\nMEM_USAGE=68%\nDISK_FREE=12GB\nUPTIME=30d 12h 45m`,
    'DSP CLOCKSTATUS': `CLOCK_SOURCE=GPS STATUS=LOCKED ACCURACY=<10ns GPS_SATELLITES=8`,
    'RST BTS': `RETCODE=0\nBTS RESET COMMAND ACCEPTED`,
    'LST ALMAF': `No active alarm filters configured`,
    'LST ALMHIS': `Total historical alarms: 156\nRecent: ALMID=20240610-001 SEVERITY=2 TIME=2024-06-10T08:15:00`,
    'CLR ALM': `RETCODE=0\nALARM CLEARED`,
    'DSP PERF': `COUNTER: RRC_CONN_ESTAB_ATT\nVALUE: 12345\nPERIOD: 15min`,
    'LST PM': `PM_RECORDS: 240\nPERIOD: 15min\nSTART: 2024-06-10T00:00:00\nEND: 2024-06-10T23:45:00`,
    'DSP RRUINFO': `RRUID=0 TYPE=AAU5239 STATUS=NORMAL TEMP=35C VSWR=1.2`,
    'DSP LINKSTATUS': `LINKTYPE=S1 STATUS=ACTIVE PEER=192.168.100.1\nLINKTYPE=X2 PEER=ENB00002 STATUS=ACTIVE`,
    'DSP SCTP': `LNKID=0 STATUS=ACTIVE STREAMS=5 HEARTBEAT=10s`,
    'LST IPADDR': `IFNAME=ETH0 IPADDR=10.1.1.100 MASK=255.255.0.0\nIFNAME=ETH1 IPADDR=192.168.200.50`,
    'LST PKG': `PKGID=V100R011C10SPC200 TYPE=FULL SIZE=512MB\nPKGID=V100R011C10SPC100 TYPE=PATCH SIZE=32MB`,
    'UPG PKG': `RETCODE=0\nUPGRADE TASK CREATED`,
  };

  const cmdPrefix = commandCode.split(' ').slice(0, 2).join(' ');
  const rawOutput = outputs[commandCode] ?? outputs[cmdPrefix] ?? `RETCODE=0 SUCCESS\n命令 ${commandCode} 执行成功`;

  return {
    success: Math.random() > 0.05,
    rawOutput,
    parsedData: { command: commandCode, device: deviceSn, status: 'ok' },
    executionTime: Math.floor(Math.random() * 2000 + 500),
    timestamp: new Date().toISOString(),
  };
}

export const mmlService = {
  async getCommands(params: { keyword?: string; category?: string } & PageRequest): Promise<PageResponse<MMLCommand>> {
    await delay(80, 150);
    let filtered = [...mockMMLCommands];
    if (params.keyword) {
      filtered = filtered.filter(
        (c) => c.commandName.includes(params.keyword!) || c.commandCode.includes(params.keyword!)
      );
    }
    if (params.category) {
      filtered = filtered.filter((c) => c.category === params.category);
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getAllCommands(): Promise<MMLCommand[]> {
    await delay(80, 150);
    return mockMMLCommands;
  },

  async executeCommand(
    commandCodeOrPayload: string | Record<string, unknown>,
    deviceSns?: string[],
    params?: Record<string, unknown>
  ): Promise<MMLTask> {
    const payload =
      typeof commandCodeOrPayload === 'string'
        ? {
            command_code: commandCodeOrPayload,
            device_sns: deviceSns ?? [],
            parameters: params,
          }
        : commandCodeOrPayload;
    const commandCode = (payload.command_code as string | undefined) ?? '';
    const requestDeviceSns = Array.isArray(payload.device_sns) ? (payload.device_sns as string[]) : [];
    const requestParameters = payload.parameters && typeof payload.parameters === 'object'
      ? (payload.parameters as Record<string, unknown>)
      : undefined;
    const fullCommand = requestParameters
      ? `${commandCode} ${Object.entries(requestParameters)
          .map(([k, v]) => `${k}:${v}`)
          .join(' ')}`
      : commandCode;
    await delay(500, 2000);
    const results = requestDeviceSns.map((sn) => ({
      deviceSn: sn,
      result: generateMMLOutput(fullCommand, sn),
    }));
    const allSuccess = results.every((r) => r.result.success);
    const newItem: MMLTask = {
      id: generateId('mmltask'),
      taskName: `Execute ${commandCode}`,
      deviceSns: requestDeviceSns,
      commands: [commandCode],
      status: 'completed',
      results,
      creator: 'admin',
      taskOrigin: 'console',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      executeType: 'immediate',
      offlineRetry: false,
      offlineRetryWait: 60,
      failedRetry: false,
      failedRetryCount: 3,
      failedRetryInterval: 5,
      totalDevices: requestDeviceSns.length,
      successCount: results.filter((r) => r.result.success).length,
      failedCount: results.filter((r) => !r.result.success).length,
      result: allSuccess ? 'success' : results.some((r) => r.result.success) ? 'partial' : 'failed',
    };
    tasks.push(newItem);
    return newItem;
  },

  async getScripts(
    p: PageRequest & { search?: string; deviceType?: string; creator?: string }
  ): Promise<PageResponse<MMLScript>> {
    await delay(80, 150);
    let filtered = scripts;
    if (p.search) {
      const kw = p.search.toLowerCase();
      filtered = filtered.filter(
        (s) =>
          s.scriptName.toLowerCase().includes(kw) ||
          (s.description ?? '').toLowerCase().includes(kw)
      );
    }
    // T-0129 batch B: MMLScript 已无 deviceType 字段 (mml.ts 移除), p.deviceType filter 失效跳过
    if (p.creator) filtered = filtered.filter((s) => s.creator === p.creator);
    return paginate(filtered, p.page, p.pageSize);
  },

  async getScriptById(id: string): Promise<MMLScript | null> {
    await delay(80, 150);
    return scripts.find((s) => s.id === id) ?? null;
  },

  async createScript(data: Omit<MMLScript, 'id' | 'createTime' | 'updateTime'>): Promise<MMLScript> {
    await delay(200, 400);
    const newItem: MMLScript = {
      ...data,
      id: generateId('script'),
      createTime: new Date().toISOString(),
      updateTime: new Date().toISOString(),
    };
    scripts.push(newItem);
    return newItem;
  },

  async updateScript(id: string, data: Partial<MMLScript>): Promise<MMLScript> {
    await delay(150, 300);
    const idx = scripts.findIndex((s) => s.id === id);
    if (idx === -1) throw new Error(`Script ${id} not found`);
    scripts[idx] = { ...scripts[idx], ...data, updateTime: new Date().toISOString() };
    return scripts[idx];
  },

  async deleteScripts(ids: string[]): Promise<void> {
    await delay(150, 300);
    scripts = scripts.filter((s) => !ids.includes(s.id));
  },

  async startScript(id: string): Promise<MMLScript> {
    await delay(80, 150);
    const idx = scripts.findIndex((s) => s.id === id);
    if (idx === -1) throw new Error(`Script ${id} not found`);
    scripts[idx] = {
      ...scripts[idx],
      status: 'running',
      startTime: new Date().toISOString(),
      endTime: undefined,
      progress: 0,
    };
    return scripts[idx];
  },

  async pauseScript(id: string): Promise<MMLScript> {
    await delay(80, 150);
    const idx = scripts.findIndex((s) => s.id === id);
    if (idx === -1) throw new Error(`Script ${id} not found`);
    scripts[idx] = { ...scripts[idx], status: 'paused' };
    return scripts[idx];
  },

  async cancelScript(id: string): Promise<MMLScript> {
    await delay(80, 150);
    const idx = scripts.findIndex((s) => s.id === id);
    if (idx === -1) throw new Error(`Script ${id} not found`);
    scripts[idx] = {
      ...scripts[idx],
      status: 'cancelled',
      endTime: new Date().toISOString(),
    };
    return scripts[idx];
  },

  async getTasks(
    p: PageRequest & { status?: string; executeType?: string; result?: string; taskName?: string; taskOrigin?: string }
  ): Promise<PageResponse<MMLTask>> {
    await delay(80, 150);
    let filtered = tasks;
    if (p.status) filtered = filtered.filter((t) => t.status === p.status);
    if (p.executeType) filtered = filtered.filter((t) => t.executeType === p.executeType);
    if (p.result) filtered = filtered.filter((t) => t.result === p.result);
    if (p.taskOrigin) filtered = filtered.filter((t) => (t.taskOrigin || (t.scriptId ? 'script' : 'console')) === p.taskOrigin);
    if (p.taskName) {
      const kw = p.taskName.toLowerCase();
      filtered = filtered.filter((t) => t.taskName.toLowerCase().includes(kw));
    }
    return paginate(filtered, p.page, p.pageSize);
  },

  async createTask(data: MMLTaskCreateInput): Promise<MMLTask> {
    await delay(200, 400);
    const planItems = data.planItems ?? [];
    const deviceSns = data.deviceSns?.length
      ? data.deviceSns
      : Array.from(new Set(planItems.map((p) => p.deviceSn).filter(Boolean)));
    const commandInputs = data.commands?.length
      ? data.commands
      : planItems.map((p) => p.command);
    const commands = commandInputs.map((cmd) => (typeof cmd === 'string' ? cmd : cmd.commandCode));
    const newItem: MMLTask = {
      ...data,
      deviceSns,
      commands,
      executeMode: data.executeMode || (planItems.length > 0 ? 'device_bound' : 'common'),
      planItems: planItems.length > 0 ? planItems : undefined,
      planStats:
        planItems.length > 0
          ? {
              totalPlanItems: planItems.length,
              totalDevices: deviceSns.length,
            }
          : undefined,
      id: generateId('mmltask'),
      status: data.executeType === 'suspended' ? 'paused' : 'pending',
      results: [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      creator: data.creator || 'admin',
      taskOrigin: data.scriptId ? 'script' : 'console',
      executeType: data.executeType || 'immediate',
      offlineRetry: data.offlineRetry ?? false,
      offlineRetryWait: data.offlineRetryWait ?? 60,
      failedRetry: data.failedRetry ?? false,
      failedRetryCount: data.failedRetryCount ?? 3,
      failedRetryInterval: data.failedRetryInterval ?? 5,
      totalDevices: deviceSns.length,
      successCount: data.successCount ?? 0,
      failedCount: data.failedCount ?? 0,
    };
    tasks.push(newItem);
    return newItem;
  },

  async executeScript(
    scriptId: string,
    deviceSns: string[]
  ): Promise<MMLTask> {
    await delay(300, 600);
    const script = scripts.find((s) => s.id === scriptId);
    if (!script) throw new Error(`Script ${scriptId} not found`);
    const commands = script.content.split('\n').filter(Boolean);
    return mmlService.createTask({
      taskName: `执行脚本: ${script.scriptName}`,
      scriptId,
      deviceSns,
      commands,
      creator: 'admin',
      executeType: 'immediate',
      offlineRetry: false,
      offlineRetryWait: 60,
      failedRetry: false,
      failedRetryCount: 3,
      failedRetryInterval: 5,
      totalDevices: deviceSns.length,
      successCount: 0,
      failedCount: 0,
    });
  },

  async getTaskById(id: string): Promise<MMLTask | null> {
    await delay(80, 150);
    return tasks.find((t) => t.id === id) ?? null;
  },

  // P4 C11 mock：按 scriptId 过滤 tasks。mock 场景下 task.scriptId 可能为空，
  // 返回空结构便于前端渲染空状态。
  async getScriptRuns(
    scriptId: string,
    p: PageRequest
  ): Promise<PageResponse<MMLTask>> {
    await delay(60, 120);
    const matched = tasks.filter((t) => t.scriptId === scriptId);
    const start = (p.page - 1) * p.pageSize;
    return {
      items: matched.slice(start, start + p.pageSize),
      total: matched.length,
      page: p.page,
      pageSize: p.pageSize,
    };
  },

  async startTask(id: string): Promise<MMLTask> {
    await delay(100, 200);
    const task = tasks.find((t) => t.id === id);
    if (!task) throw new Error(`Task ${id} not found`);
    if (task.status !== 'pending' && task.status !== 'paused') {
      throw new Error(`Cannot start task in ${task.status} state`);
    }
    task.status = 'running';
    return task;
  },

  async startTasks(ids: string[]): Promise<void> {
    for (const id of ids) {
      await mmlService.startTask(id);
    }
  },

  async pauseTask(id: string): Promise<MMLTask> {
    await delay(100, 200);
    const task = tasks.find((t) => t.id === id);
    if (!task) throw new Error(`Task ${id} not found`);
    if (task.status !== 'running') {
      throw new Error(`Cannot pause task in ${task.status} state`);
    }
    task.status = 'paused';
    return task;
  },

  async cancelTask(id: string): Promise<MMLTask> {
    await delay(100, 200);
    const task = tasks.find((t) => t.id === id);
    if (!task) throw new Error(`Task ${id} not found`);
    task.status = 'cancelled';
    return task;
  },

  async cancelTasks(ids: string[]): Promise<void> {
    await delay(100, 200);
    tasks = tasks.map((task) => (
      ids.includes(task.id) ? { ...task, status: 'cancelled' } : task
    ));
  },

  async deleteTask(id: string): Promise<void> {
    await delay(100, 200);
    tasks = tasks.filter((t) => t.id !== id);
  },

  async deleteTasks(ids: string[]): Promise<void> {
    await delay(100, 200);
    tasks = tasks.filter((t) => !ids.includes(t.id));
  },

  async getTaskResults(
    id: string,
    page = 1,
    pageSize = 20
  ): Promise<PageResponse<DeviceTaskResultItem, MMLTaskResultsStats>> {
    await delay(80, 150);
    const task = tasks.find((t) => t.id === id);
    const results = task?.results ?? [];
    return {
      items: results.slice((page - 1) * pageSize, page * pageSize),
      total: results.length,
      page,
      pageSize,
    };
  },

  async checkDangerous(
    commandCode: string
  ): Promise<{ dangerous: boolean; info: { Name: string; Desc: string } | null }> {
    await delay(30, 80);
    const dangerousPatterns = [
      { pattern: /\bRST\b/i, name: '重启', desc: '此操作将重启设备' },
      { pattern: /\bFACTORYRESET\b/i, name: '恢复默认配置', desc: '此操作将恢复设备出厂设置' },
      { pattern: /\bCOLDREBOOT\b/i, name: '冷重启', desc: '此操作将执行设备冷重启' },
    ];
    for (const dp of dangerousPatterns) {
      if (dp.pattern.test(commandCode)) {
        return { dangerous: true, info: { Name: dp.name, Desc: dp.desc } };
      }
    }
    return { dangerous: false, info: null };
  },

  // --- Templates (mock) ---

  async getTemplates(
    params?: { commandCode?: string; operationType?: string; templateScope?: string } & PageRequest
  ): Promise<PageResponse<MMLCustomCommand>> {
    await delay(80, 150);
    return {
      items: [],
      total: 0,
      page: params?.page ?? 1,
      pageSize: params?.pageSize ?? 20,
    };
  },

  async createTemplate(
    _tmpl: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'>
  ): Promise<MMLCustomCommand> {
    await delay(100, 200);
    return {
      id: generateId('mmltmpl'),
      ..._tmpl,
      creator: 'admin',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
  },

  async updateTemplate(
    _id: string,
    _tmpl: Partial<MMLCustomCommand>
  ): Promise<MMLCustomCommand> {
    await delay(100, 200);
    return {
      id: _id,
      commandName: _tmpl.commandName ?? '',
      commandCode: _tmpl.commandCode ?? '',
      operationType: _tmpl.operationType ?? 'LST',
      commandScope: _tmpl.commandScope ?? 'private',
      categoryGroup: _tmpl.categoryGroup ?? '',
      parameters: _tmpl.parameters ?? {},
      paramPaths: _tmpl.paramPaths ?? [],
      description: _tmpl.description ?? '',
      creator: 'admin',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
  },

  async deleteTemplate(_id: string): Promise<void> {
    await delay(100, 200);
  },

  async cloneTemplate(_id: string): Promise<MMLCustomCommand> {
    await delay(100, 200);
    return {
      id: generateId('mmltmpl'),
      commandName: 'Cloned Command',
      commandCode: 'LST CELL',
      operationType: 'LST',
      commandScope: 'private',
      categoryGroup: '',
      parameters: {},
      paramPaths: [],
      description: '',
      creator: 'admin',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
  },
};
