import { describe, expect, it } from 'vitest';

import { mockStandardParams } from '../../data/paramModel';
import { paramModelService } from '../paramModelService';

describe('paramModelService.createStandard', () => {
  it('rejects an existing standard path instead of overwriting or duplicating it', async () => {
    const existing = mockStandardParams[0];

    await expect(
      paramModelService.createStandard({
        ...existing,
        access: existing.access === 'readOnly' ? 'readWrite' : 'readOnly',
      }),
    ).rejects.toThrow(`参数 path "${existing.standardPath}" 已存在，只能在原有记录上修改`);

    const result = await paramModelService.listStandard({ keyword: existing.standardPath });
    expect(result.items.filter((item) => item.standardPath === existing.standardPath)).toHaveLength(1);
    expect(result.items[0].access).toBe(existing.access);
  });
});
