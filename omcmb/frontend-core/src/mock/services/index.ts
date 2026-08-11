export { deviceService } from './deviceService';
export { alarmService } from './alarmService';
export { neService } from './neService';
export { performanceService } from './performanceService';
export { configService } from './configService';
export { topologyService } from './topologyService';
export { mmlService } from './mmlService';
export { backupService } from './backupService';
// licenseService 已在 F06 System License 重构 Step 5 删除（multi-license 模型下线）。
// 新 system_license API 没有 mock 实现，详见 hooks/api/useSystemLicense.ts。
export { softwareService } from './softwareService';
export { fileService } from './fileService';
export { logService } from './logService';
export { reportService } from './reportService';
export { mrService } from './mrService';
export { systemService } from './systemService';
export { opsToolsService } from './opsToolsService';
export { dashboardService } from './dashboardService';
export { indicatorService } from './indicatorService';
export { notificationCenterService } from './notificationCenterService';
export {
  geofenceMockService,
  resetGeofenceMock,
} from './geofenceMock';
