import { readFileSync } from 'node:fs';

import { describe, expect, it } from 'vitest';

describe('UserDropdown change-password modal', () => {
  it('identifies the current signed-in account without using user-management selection state', () => {
    const source = readFileSync(new URL('./UserDropdown.tsx', import.meta.url), 'utf8');
    const modal = source.slice(source.indexOf('{/* 修改密码 Modal'), source.indexOf('</Modal>'));

    expect(modal).toContain("t('user.currentAccount', { username: currentUser?.username ?? '-' })");
    expect(modal).not.toContain('selectedUser');
  });
});
