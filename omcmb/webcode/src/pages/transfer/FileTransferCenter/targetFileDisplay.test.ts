import { describe, expect, it } from 'vitest';
import { resolveTargetFileDisplay } from './targetFileDisplay';

describe('resolveTargetFileDisplay', () => {
  it('keeps quota-cleaned file names visible but not downloadable', () => {
    const state = resolveTargetFileDisplay(
      {
        targetFile: ' ErrorLog_20260715.1432 0800_dieLog(1).tar.gz ',
        downloadUrl: 'http://example.invalid/download',
        fileDeleted: true,
      },
      '已被配额清理',
    );

    expect(state).toEqual({
      file: 'ErrorLog_20260715.1432 0800_dieLog(1).tar.gz',
      downloadUrl: '',
      tooltip: '已被配额清理',
      deleted: true,
    });
  });

  it('keeps active uploaded files clickable', () => {
    const state = resolveTargetFileDisplay(
      {
        targetFile: 'runtime-deadbeef-SN001.tar.gz',
        downloadUrl: 'http://example.invalid/runtime',
      },
      '已被配额清理',
    );

    expect(state).toEqual({
      file: 'runtime-deadbeef-SN001.tar.gz',
      downloadUrl: 'http://example.invalid/runtime',
      tooltip: 'runtime-deadbeef-SN001.tar.gz',
      deleted: false,
    });
  });
});
