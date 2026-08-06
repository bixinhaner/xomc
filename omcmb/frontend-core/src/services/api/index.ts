/**
 * API Service Layer — Entry point.
 *
 * Each backend domain will have its own API module file here.
 *
 * Sprint 1: authApi, deviceApi, alarmApi ✅
 * Sprint 2: templateApi, softwareApi, adminApi, topologyApi ✅
 * Sprint 3: pmApi, mrApi ✅
 * Sprint 4: dashboardApi, logApi, northboundApi, provisionApi, interopApi ✅
 * Sprint 9: backupApi, fileApi, mmlApi ✅
 * T-0098 P4: paramModelApi, productApi, indicatorLibraryApi, alarmDefinitionApi ✅
 * T-0098 P5: 移除 datamodelApi（被 paramModelApi + productApi 取代）✅
 */

export { http } from '../http';
export { authApi } from './authApi';
export { agentApi } from './agentApi';
export { deviceApi } from './deviceApi';
export { alarmApi } from './alarmApi';
export { templateApi } from './templateApi';
export { softwareApi } from './softwareApi';
export { adminApi } from './adminApi';
export { topologyApi } from './topologyApi';
export { pmApi } from './pmApi';
export { mrApi } from './mrApi';
export { dashboardApi } from './dashboardApi';
export { logApi } from './logApi';
export { northboundApi } from './northboundApi';
export { provisionApi } from './provisionApi';
export { interopApi } from './interopApi';
export { configSyncApi } from './configSyncApi';
export { backupApi } from './backupApi';
export { fileApi } from './fileApi';
export { mmlApi } from './mmlApi';
export { traceApi } from './traceApi';
export { geofenceApi } from './geofenceApi';
