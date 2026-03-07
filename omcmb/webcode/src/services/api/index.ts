/**
 * API Service Layer — Entry point.
 *
 * Each backend domain will have its own API module file here.
 * These will be implemented progressively starting from Sprint 1:
 *
 * Sprint 1: authApi, deviceApi, alarmApi ✅
 * Sprint 2: templateApi, softwareApi, adminApi, topologyApi ✅
 * Sprint 3: pmApi, mrApi ✅
 * Sprint 4: dashboardApi, logApi, datamodelApi, northboundApi, provisionApi, interopApi ✅
 * Sprint 9: backupApi, fileApi, mmlApi ✅
 */

export { http } from '../http';
export { authApi } from './authApi';
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
export { datamodelApi } from './datamodelApi';
export { northboundApi } from './northboundApi';
export { provisionApi } from './provisionApi';
export { interopApi } from './interopApi';
export { configSyncApi } from './configSyncApi';
export { backupApi } from './backupApi';
export { fileApi } from './fileApi';
export { mmlApi } from './mmlApi';
