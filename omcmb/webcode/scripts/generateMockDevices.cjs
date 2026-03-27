/**
 * 生成 GIS 地图 Mock 设备数据
 * 解析 CSV 文件并转换为 TypeScript 格式
 */

const fs = require('fs');
const path = require('path');

// CSV 文件路径
const gnbCsvPath = path.resolve(__dirname, '../../../files/Front-end/GNB 20260210051225.csv');
const stationCsvPath = path.resolve(__dirname, '../../../files/Front-end/Station_20260210050140.csv');
const outputPath = path.resolve(__dirname, '../src/components/GISMap/mockDeviceData.ts');

/**
 * 解析 CSV 行（处理引号内的逗号和制表符）
 */
function parseCSVLine(line) {
  const result = [];
  let current = '';
  let inQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    if (char === '"') {
      inQuotes = !inQuotes;
    } else if ((char === ',' || char === '\t') && !inQuotes) {
      result.push(current.trim());
      current = '';
    } else {
      current += char;
    }
  }
  result.push(current.trim());
  return result;
}

/**
 * 解析 GNB CSV 文件 (5G NR 设备)
 */
function parseGNBCsv(content) {
  const lines = content.split('\n').filter(line => line.trim());
  const devices = [];

  // 跳过标题行
  for (let i = 1; i < lines.length; i++) {
    const cols = parseCSVLine(lines[i]);
    if (cols.length < 32) continue;

    const status = cols[0]?.toLowerCase().trim();
    const alarmCount = parseInt(cols[1]?.trim()) || 0;
    const serialNumber = cols[2]?.trim();
    const gnbName = cols[6]?.trim();
    const deviceGroup = cols[13]?.trim();
    const siteName = cols[34]?.trim() || cols[33]?.trim() || '';
    const longitude = parseFloat(cols[30]?.trim());
    const latitude = parseFloat(cols[31]?.trim());

    if (isNaN(longitude) || isNaN(latitude) || !gnbName) continue;

    devices.push({
      id: `gnb-${i}`,
      name: gnbName,
      sn: serialNumber,
      lng: longitude,
      lat: latitude,
      status: status === 'on' ? 'online' : 'offline',
      alarmCount: alarmCount,
      groupName: deviceGroup,
      address: siteName,
      type: 'macro'
    });
  }

  return devices;
}

/**
 * 解析 Station CSV 文件 (4G LTE 设备)
 * 列索引（从0开始）: Online Status(0), Alarm Count(1), Serial Number(2), Cell Name(3),
 *                   Device Group(16), Longitude(19), Latitude(20)
 */
function parseStationCsv(content) {
  const lines = content.split('\n').filter(line => line.trim());
  const devices = [];

  // 跳过标题行
  for (let i = 1; i < lines.length; i++) {
    const cols = parseCSVLine(lines[i]);
    if (cols.length < 21) continue;

    const status = cols[0]?.toLowerCase().trim();
    const alarmCount = parseInt(cols[1]?.trim()) || 0;
    const serialNumber = cols[2]?.trim();
    const cellName = cols[3]?.trim();
    const deviceGroup = cols[16]?.trim();
    const longitude = parseFloat(cols[19]?.trim());
    const latitude = parseFloat(cols[20]?.trim());

    if (isNaN(longitude) || isNaN(latitude) || !cellName) continue;
    // 过滤掉无效坐标（经度115在中国，不是非洲赞比亚）
    if (longitude > 50 || longitude < -50) continue;

    devices.push({
      id: `lte-${i}`,
      name: cellName,
      sn: serialNumber,
      lng: longitude,
      lat: latitude,
      status: status === 'online' ? 'online' : 'offline',
      alarmCount: alarmCount,
      groupName: deviceGroup,
      address: '',
      type: 'small'
    });
  }

  return devices;
}

// 主函数
function main() {
  console.log('开始解析 CSV 文件...');

  // 读取 GNB CSV
  const gnbContent = fs.readFileSync(gnbCsvPath, 'utf-8');
  const gnbDevices = parseGNBCsv(gnbContent);
  console.log(`解析 GNB 数据: ${gnbDevices.length} 条设备`);

  // 读取 Station CSV
  const stationContent = fs.readFileSync(stationCsvPath, 'utf-8');
  const stationDevices = parseStationCsv(stationContent);
  console.log(`解析 Station 数据: ${stationDevices.length} 条设备`);

  // 合并数据
  const allDevices = [...gnbDevices, ...stationDevices];
  console.log(`总计: ${allDevices.length} 条设备`);

  // 生成 TypeScript 文件内容
  const tsContent = `/**
 * GIS 地图 Mock 设备数据
 * 自动生成自 CSV 文件
 * 数据来源: GNB 20260210051225.csv + Station_20260210050140.csv
 * 生成时间: ${new Date().toISOString()}
 * 设备总数: ${allDevices.length}
 */

import type { MapDevice } from '@/types/map';

export const mockMapDevices: MapDevice[] = ${JSON.stringify(allDevices, null, 2)};

// 统计信息
export const mockStats = {
  total: ${allDevices.length},
  online: ${allDevices.filter(d => d.status === 'online').length},
  offline: ${allDevices.filter(d => d.status === 'offline').length},
  alarms: ${allDevices.reduce((sum, d) => sum + (d.alarmCount || 0), 0)},
};
`;

  // 写入文件
  fs.writeFileSync(outputPath, tsContent, 'utf-8');
  console.log(`\n生成完成: ${outputPath}`);
  console.log(`文件大小: ${(fs.statSync(outputPath).size / 1024).toFixed(2)} KB`);
}

main();
