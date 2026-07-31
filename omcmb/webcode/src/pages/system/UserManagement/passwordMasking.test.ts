import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

describe('UserManagement reset-password fields', () => {
  it('keeps both password fields masked without visibility toggles', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/pages/system/UserManagement/index.tsx'), 'utf8');
    const resetModal = source.slice(
      source.indexOf('{/* Reset Password Modal */}'),
      source.indexOf('{/* 批量分配角色 Modal'),
    );

    expect(resetModal.match(/visibilityToggle=\{false\}/g)).toHaveLength(2);
    expect(resetModal.match(/autoComplete="new-password"/g)).toHaveLength(2);
  });
});
