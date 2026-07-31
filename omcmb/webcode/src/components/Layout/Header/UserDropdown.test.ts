import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

describe('UserDropdown change-password modal', () => {
  it('identifies the current signed-in account without using user-management selection state', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/components/Layout/Header/UserDropdown.tsx'), 'utf8');
    const modal = source.slice(source.indexOf('{/* 修改密码 Modal'), source.indexOf('</Modal>'));

    expect(modal).toContain("t('user.currentAccount', { username: currentUser?.username ?? '-' })");
    expect(modal).not.toContain('selectedUser');
  });
});
