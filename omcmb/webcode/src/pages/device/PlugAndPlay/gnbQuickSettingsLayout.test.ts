import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const pageSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/AddPolicyPage.tsx'),
  'utf8',
);
const gnbSection = pageSource.slice(
  pageSource.indexOf('{/* gNB specific fields */}'),
  pageSource.indexOf('{/* GSM specific fields */}'),
);

function visualOrder(key: string): number {
  const match = gnbSection.match(new RegExp(`key="${key}"[\\s\\S]*?order: (\\d+)`));
  if (!match) throw new Error(`missing gNB parameter section: ${key}`);
  return Number(match[1]);
}

describe('gNB parameter editor quick-settings layout', () => {
  it('uses the same card and three-column layout as quick settings', () => {
    expect(gnbSection).not.toContain('<Collapse.Panel');
    expect(gnbSection).toContain('<GnbQuickSettingsCards />');
    expect(gnbSection).toContain("gridTemplateColumns: 'repeat(3, minmax(0, 1fr))'");
  });

  it('places quick-setting sections through TDD before template-only sections', () => {
    expect(gnbSection).toContain('<Form.List name="plmnConfigList">');
    expect(['gnb-other', 'gnb-plmn-extra', 'gnb-network', 'gnb-slice', 'gnb-custom'].map(visualOrder))
      .toEqual([8, 9, 10, 11, 12]);
  });
});
