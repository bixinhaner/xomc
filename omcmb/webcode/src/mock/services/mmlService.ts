import type { MMLCommand, MMLScript, MMLTask, MMLResult } from '@/types/mml';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { mockMMLCommands, mockMMLScripts, mockMMLTasks } from '../data/mml';
import { delay, paginate, generateId } from '../utils';

let scripts = [...mockMMLScripts];
let tasks = [...mockMMLTasks];

function generateMMLOutput(commandCode: string, deviceSn: string): MMLResult {
  const outputs: Record<string, string> = {
    'DSP VERSION': `VERSION: V100R011C10SPC200\nDEVICE: ${deviceSn}\nBUILD: 20240315\nSTATUS: Normal`,
    'DSP BOARDSTATUS': `SRN=0 SN=0 TYPE=BBU STATUS=NORMAL\nSRN=0 SN=1 TYPE=UMPT STATUS=NORMAL\nSRN=0 SN=2 TYPE=UBBP STATUS=NORMAL`,
    'LST CELL': `CELLID=0 STATUS=ACTIVE ADMINSTATE=UNLOCKED PCI=128 TXPOWER=43\nCELLID=1 STATUS=ACTIVE ADMINSTATE=UNLOCKED PCI=129 TXPOWER=43`,
    'DSP SYSRESOURCE': `CPU_USAGE=45%\nMEM_USAGE=68%\nDISK_FREE=12GB\nUPTIME=30d 12h 45m`,
    'DSP CLOCKSTATUS': `CLOCK_SOURCE=GPS STATUS=LOCKED ACCURACY=<10ns GPS_SATELLITES=8`,
    'DSP LINKSTATUS': `LINKTYPE=S1 STATUS=ACTIVE PEER=192.168.100.1\nLINKTYPE=X2 PEER=ENB00002 STATUS=ACTIVE`,
    'LST ALMAF': `No active alarm filters configured`,
    'DSP SCTP': `LNKID=0 STATUS=ACTIVE STREAMS=5 HEARTBEAT=10s`,
    'LST IPADDR': `IFNAME=ETH0 IPADDR=10.1.1.100 MASK=255.255.0.0\nIFNAME=ETH1 IPADDR=192.168.200.50`,
    'DSP RRUINFO': `RRUID=0 TYPE=AAU5239 STATUS=NORMAL TEMP=35C VSWR=1.2`,
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
    commandCode: string,
    deviceSns: string[],
    params?: Record<string, string | number | boolean>
  ): Promise<Array<{ deviceSn: string; result: MMLResult }>> {
    const fullCommand = params
      ? `${commandCode} ${Object.entries(params)
          .map(([k, v]) => `${k}:${v}`)
          .join(' ')}`
      : commandCode;
    await delay(500, 2000);
    return deviceSns.map((sn) => ({
      deviceSn: sn,
      result: generateMMLOutput(fullCommand, sn),
    }));
  },

  async getScripts(p: PageRequest): Promise<PageResponse<MMLScript>> {
    await delay(80, 150);
    return paginate(scripts, p.page, p.pageSize);
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

  async getTasks(p: PageRequest): Promise<PageResponse<MMLTask>> {
    await delay(80, 150);
    return paginate(tasks, p.page, p.pageSize);
  },

  async createTask(data: Omit<MMLTask, 'id' | 'status' | 'results' | 'createdAt' | 'updatedAt'>): Promise<MMLTask> {
    await delay(200, 400);
    const newItem: MMLTask = {
      ...data,
      id: generateId('mmltask'),
      status: 'pending',
      results: [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
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
    });
  },
};
