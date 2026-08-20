import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { normalizeOriginalVersionsForSubmit } from './softwareUpgradeOriginalVersion';

const pageSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/AddPolicyPage.tsx'),
  'utf8',
);

describe('software upgrade original version selector', () => {
  it('keeps selected and manually-entered versions as an editable list', () => {
    expect(normalizeOriginalVersionsForSubmit('specify', [
      ' BaiBNQ_2.9.0.6 ',
      'BaiBNQ_2.9.0.7',
      'BaiBNQ_2.9.0.6',
    ])).toEqual(['BaiBNQ_2.9.0.6', 'BaiBNQ_2.9.0.7']);
    expect(normalizeOriginalVersionsForSubmit('specify', 'V1.0,V1.1；V1.2')).toEqual([
      'V1.0',
      'V1.1',
      'V1.2',
    ]);
  });

  it('persists all-version selection explicitly', () => {
    expect(normalizeOriginalVersionsForSubmit('all', ['V1.0'])).toEqual(['all']);
  });

  it('uses tags mode so operators can type versions not in the detected list', () => {
    expect(pageSource).toContain('mode="tags"');
    expect(pageSource).toContain("tokenSeparators={[',', '，', ';', '；', '\\n']}");
    expect(pageSource).toContain('originalVersion: submittedOriginalVersion');
  });
});
