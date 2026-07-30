import { describe, expect, it } from 'vitest';

import { mockStandardParams } from '../../data/paramModel';
import { paramModelService } from '../paramModelService';

describe('paramModelService.createStandard', () => {
  it('rejects an existing standard path instead of overwriting or duplicating it', async () => {
    const existing = mockStandardParams[0];

    const error = await paramModelService.createStandard({
      ...existing,
      access: existing.access === 'readOnly' ? 'readWrite' : 'readOnly',
    }).catch((err: unknown) => err as Error & { bizCode?: number });

    expect(error).toMatchObject({
      bizCode: 2034,
      message: `standard path "${existing.standardPath}" already exists; update the existing record instead`,
    });
    expect(
      paramModelService.createStandard({
        ...existing,
        access: existing.access === 'readOnly' ? 'readWrite' : 'readOnly',
      }),
    ).rejects.toMatchObject({ bizCode: 2034 });

    const result = await paramModelService.listStandard({ keyword: existing.standardPath });
    expect(result.items.filter((item) => item.standardPath === existing.standardPath)).toHaveLength(1);
    expect(result.items[0].access).toBe(existing.access);
  });
});
