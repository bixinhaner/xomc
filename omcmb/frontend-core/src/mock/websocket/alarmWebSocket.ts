import type { Alarm } from '../../types/alarm';
import type { AlarmSeverity } from '../../types/common';
import { MockWebSocket } from './index';
import { mockActiveAlarms } from '../data/alarms';
import { mockDevices } from '../data/devices';
import { useAlarmStore } from '../../store/alarmStore';

const alarmNames = [
  '小区不可用', 'S1链路中断', '射频单元故障', '温度过高告警', 'VSWR异常',
  'GPS失锁', 'X2链路故障', '传输中断', '电压异常', 'CPU过载告警',
  'E-RAB建立失败', '切换成功率低', '上行干扰过高', '下行吞吐量低', '内存利用率高',
  '风扇故障', '光模块异常', 'NTP同步失败', '基带板故障', 'RRU掉线',
];

const alarmTypes = ['无线告警', '传输告警', '硬件告警', '环境告警', '射频告警', '时钟告警', '系统告警', '核心网告警'];

const severityWeights: Array<[AlarmSeverity, number]> = [
  ['critical', 0.08],
  ['major', 0.22],
  ['minor', 0.30],
  ['warning', 0.40],
];

function pickWeightedSeverity(): AlarmSeverity {
  const rand = Math.random();
  let cumulative = 0;
  for (const [severity, weight] of severityWeights) {
    cumulative += weight;
    if (rand < cumulative) return severity;
  }
  return 'warning';
}

function pickRandom<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

let alarmIdCounter = 10000;

function generateNewAlarm(): Alarm {
  const device = pickRandom(mockDevices.filter((d) => d.connStatus === 'online'));
  const alarmName = pickRandom(alarmNames);
  const severity = pickWeightedSeverity();
  const alarmType = pickRandom(alarmTypes);
  const codeNum = String(Math.floor(Math.random() * 200) + 1).padStart(4, '0');

  const id = `WS-ALM-${String(++alarmIdCounter).padStart(8, '0')}`;

  return {
    id,
    alarmCode: `A${codeNum}`,
    alarmName,
    severity,
    deviceSn: device.sn,
    deviceName: device.name,
    description: `设备${device.name}检测到${alarmName}，${alarmType}异常，请及时处理`,
    neType: device.productType,
    eventTime: new Date().toISOString(),
    updTime: new Date().toISOString(),
    clearTime: undefined,
    duration: 0,
    dealState: '0' as const,
    alarmType: 'active' as const,
    unread: '1' as const,
    alarmCount: 1,
    alarmSource: device.sn,
    technology: device.productType,
    isActive: true,
    equipInfo: `${device.name}(${device.sn})`,
    eventType: 'device' as const,
    specificProblem: alarmName,
  };
}

export class AlarmWebSocket extends MockWebSocket {
  private activeAlarms: Alarm[] = [...mockActiveAlarms];

  protected onConnect(): void {
    this.startInterval(() => this.simulateAlarmEvent(), 3000, 8000);
  }

  private simulateAlarmEvent(): void {
    const isNew = Math.random() < 0.7;

    if (isNew) {
      const alarm = generateNewAlarm();
      this.activeAlarms.push(alarm);
      this.emit<Alarm>('alarm_new', alarm);

      // Update alarm store counts
      try {
        const store = useAlarmStore.getState();
        store.incrementSeverity(alarm.severity);
      } catch {
        // Store may not be available in test environments
      }
    } else {
      if (this.activeAlarms.length === 0) return;
      const randomAlarm = pickRandom(this.activeAlarms);
      const clearTime = new Date().toISOString();
      const clearedAlarm: Alarm = {
        ...randomAlarm,
        clearTime,
        isActive: false,
        duration: Math.floor(
          (new Date(clearTime).getTime() - new Date(randomAlarm.eventTime).getTime()) / 60000
        ),
      };
      this.activeAlarms = this.activeAlarms.filter((a) => a.id !== randomAlarm.id);
      this.emit<{ id: string; alarm: Alarm }>('alarm_clear', { id: randomAlarm.id, alarm: clearedAlarm });

      try {
        const store = useAlarmStore.getState();
        store.decrementSeverity(randomAlarm.severity);
      } catch {
        // Store may not be available in test environments
      }
    }
  }

  getActiveAlarms(): Alarm[] {
    return [...this.activeAlarms];
  }
}

export const alarmWebSocket = new AlarmWebSocket();
