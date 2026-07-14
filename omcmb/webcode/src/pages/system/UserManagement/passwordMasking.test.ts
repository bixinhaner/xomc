import { readFileSync } from 'node:fs';

import { describe, expect, it } from 'vitest';

describe('UserManagement reset-password fields', () => {
  it('keeps both password fields masked without visibility toggles', () => {
    const source = readFileSync(new URL('./index.tsx', import.meta.url), 'utf8');
    const resetModal = source.slice(
      source.indexOf('{/* Reset Password Modal */}'),
      source.indexOf('{/* 批量分配角色 Modal'),
    );

    expect(resetModal.match(/visibilityToggle=\{false\}/g)).toHaveLength(2);
    expect(resetModal.match(/autoComplete="new-password"/g)).toHaveLength(2);
  });
});
