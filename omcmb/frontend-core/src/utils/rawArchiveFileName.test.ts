import { describe, expect, it } from 'vitest';

import { resolveRawArchiveDisplayFileName } from './rawArchiveFileName';

describe('resolveRawArchiveDisplayFileName', () => {
  it('存储对象已压缩时，展示名补齐 .gz 后缀', () => {
    expect(
      resolveRawArchiveDisplayFileName(
        'issue836-real-48BF74.1202000240194DP0015.xml',
        '2026/07/03/issue836-real-48BF74.1202000240194DP0015.xml.gz',
      ),
    ).toBe('issue836-real-48BF74.1202000240194DP0015.xml.gz');
  });

  it('展示名已有 .gz 时不重复追加', () => {
    expect(
      resolveRawArchiveDisplayFileName(
        'issue836-real-48BF74.1202000240194DP0015.xml.gz',
        '2026/07/03/issue836-real-48BF74.1202000240194DP0015.xml.gz',
      ),
    ).toBe('issue836-real-48BF74.1202000240194DP0015.xml.gz');
  });

  it('存储对象未压缩时保持原名', () => {
    expect(
      resolveRawArchiveDisplayFileName(
        'issue836-real-48BF74.1202000240194DP0015.xml',
        '2026/07/03/issue836-real-48BF74.1202000240194DP0015.xml',
      ),
    ).toBe('issue836-real-48BF74.1202000240194DP0015.xml');
  });
});
