import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const tileMetadata = {
  name: 'China Offline Map',
  bounds: [73, 3, 135, 54],
  center: [104, 35, 4],
  minzoom: 3,
  maxzoom: 15,
};

describe('useMapConfig tile availability cache', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('重新进入页面时会重试上一次失败的瓦片健康检查', async () => {
    let tileChecks = 0;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url === '/tiles-metadata') {
        return new Response(JSON.stringify(tileMetadata), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (url.startsWith('/tiles/') && init?.method === 'HEAD') {
        tileChecks += 1;
        return new Response(null, { status: tileChecks === 1 ? 404 : 200 });
      }

      throw new Error(`unexpected fetch: ${url}`);
    });
    vi.stubGlobal('fetch', fetchMock);

    const { useMapConfig } = await import('./useMapConfig');
    const firstMount = renderHook(() => useMapConfig());

    await waitFor(() => {
      expect(firstMount.result.current.status).toBe('success');
      expect(firstMount.result.current.tilesAvailable).toBe(false);
    });
    firstMount.unmount();

    const secondMount = renderHook(() => useMapConfig());
    await waitFor(() => {
      expect(secondMount.result.current.tilesAvailable).toBe(true);
    });

    expect(tileChecks).toBe(2);
    expect(fetchMock.mock.calls.filter(([url]) => String(url) === '/tiles-metadata')).toHaveLength(1);
    secondMount.unmount();
  });
});
