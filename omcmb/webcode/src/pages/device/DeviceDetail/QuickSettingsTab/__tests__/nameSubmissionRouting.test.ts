import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('quick settings name submission routing', () => {
  it('treats HNBName and gNBName as LMT parameters instead of OMC rename operations', () => {
    const source = readFileSync(resolve(
      process.cwd(),
      'src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx',
    ), 'utf8');

    expect(source).not.toContain('useRenameDevice');
    expect(source).not.toContain('renameMutation');
    expect(source).toContain('useUpdateParameters');
  });

  it('keeps the OMC rename operation in the device detail name editor', () => {
    const source = readFileSync(resolve(
      process.cwd(),
      'src/pages/device/DeviceDetail/index.tsx',
    ), 'utf8');

    expect(source).toContain('useRenameDevice');
    expect(source).toContain("t('device.nameSync.editOmcName')");
    expect(source).not.toContain("nameSyncMode !== 'auto_lmt_to_omc'");
    expect(source).not.toContain("if (!displayDevice || nameSyncMode === 'auto_lmt_to_omc') return;");
  });
});
