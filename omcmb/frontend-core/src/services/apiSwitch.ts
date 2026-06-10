/**
 * Mock/Real API switch mechanism.
 *
 * Controlled by the VITE_USE_MOCK environment variable:
 *   - `true`  → use mock services (local data, no backend required)
 *   - `false` → use real API services (requires backend running)
 *
 * Usage in hook files (starting from Sprint 1):
 *   import { createApiSwitch } from '../services/apiSwitch';
 *   import { deviceService as mockDeviceService } from '../mock/services/deviceService';
 *   import { deviceApi } from '../services/api/deviceApi';
 *   const deviceService = createApiSwitch(mockDeviceService, deviceApi);
 */

export const useMock = import.meta.env.VITE_USE_MOCK === 'true';

/**
 * Creates a service that switches between mock and real implementations
 * based on the VITE_USE_MOCK environment variable.
 *
 * Use this when mock and real services share an identical surface (same type).
 */
export function createApiSwitch<T>(mockService: T, realService: T): T {
  return useMock ? mockService : realService;
}

/**
 * Switch variant for services whose mock surface only approximates the real one.
 *
 * The real API is treated as the canonical contract; the mock is best-effort and
 * adapted at runtime. This collapses the previously inconsistent boundary casts
 * (inline `useMock ? (mock as unknown as typeof real) : real`,
 * `createApiSwitch(mock as unknown as typeof real, real)`, and plain
 * `useMock ? mock : real`) into a single typed helper so the returned `TReal`
 * type never drifts between the mock and real branches.
 *
 * Usage:
 *   const api = createApiSwitchWithMock(deviceService, deviceApi); // typed as typeof deviceApi
 */
export function createApiSwitchWithMock<TReal>(
  mockService: unknown,
  realService: TReal,
): TReal {
  return useMock ? (mockService as TReal) : realService;
}
