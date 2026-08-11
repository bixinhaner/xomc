import { describe, expect, it } from 'vitest';
import type { MapMetadata } from '@/components/GISMap/useMapConfig';
import { resolveMapInitialView } from './mapValidation';

const metadata: MapMetadata = {
  name: 'China Offline Map',
  region: 'China',
  center: { lon: 104, lat: 35, zoom: 4 },
  bounds: { minLon: 73, maxLon: 135, minLat: 3, maxLat: 54 },
  zoom: { min: 3, max: 15, default: 4 },
  attribution: 'Authorized map data',
  description: 'Test map',
};

const defaults = {
  metadata: null,
  metadataAvailable: false,
  devices: [],
  envCenter: null,
  defaultCenter: [104, 35] as [number, number],
  defaultZoom: 4,
};

describe('resolveMapInitialView', () => {
  it('优先使用有效离线地图元数据中心点', () => {
    const result = resolveMapInitialView({
      ...defaults,
      metadata,
      metadataAvailable: true,
      envCenter: { center: [28.221, -14.607], zoom: 6 },
    });

    expect(result).toEqual({ center: [104, 35], zoom: 4, source: 'metadata' });
  });

  it('离线元数据不可用时使用有效设备坐标中心点', () => {
    const result = resolveMapInitialView({
      ...defaults,
      metadata,
      metadataAvailable: false,
      devices: [
        { longitude: 116, latitude: 39 },
        { longitude: 118, latitude: 41 },
      ],
      envCenter: { center: [28.221, -14.607], zoom: 6 },
    });

    expect(result.source).toBe('device_data');
    expect(result.center).toEqual([117, 40]);
  });

  it('没有设备坐标时使用环境默认中心点', () => {
    const result = resolveMapInitialView({
      ...defaults,
      envCenter: { center: [28.221, -14.607], zoom: 6 },
    });

    expect(result).toEqual({ center: [28.221, -14.607], zoom: 6, source: 'env_config' });
  });

  it('没有离线地图、设备和环境配置时使用代码默认中心点', () => {
    expect(resolveMapInitialView(defaults)).toEqual({
      center: [104, 35],
      zoom: 4,
      source: 'default',
    });
  });

  it('无效设备坐标不会阻止继续使用环境或代码默认中心点', () => {
    const result = resolveMapInitialView({
      ...defaults,
      devices: [
        { longitude: null, latitude: 39 },
        { longitude: 116, latitude: undefined },
      ],
    });

    expect(result.source).toBe('default');
    expect(result.center).toEqual([104, 35]);
  });
});
