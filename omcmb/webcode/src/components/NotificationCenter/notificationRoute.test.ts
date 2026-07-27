import { describe, expect, it } from 'vitest';

import { buildNotificationMessageRoute } from './index';

describe('buildNotificationMessageRoute', () => {
  it('routes top-bar notification clicks to the message center detail', () => {
    expect(buildNotificationMessageRoute('msg-001')).toBe(
      '/notifications?tab=messages&messageId=msg-001',
    );
  });

  it('encodes message ids instead of reusing backend related links', () => {
    expect(buildNotificationMessageRoute('task/with space')).toBe(
      '/notifications?tab=messages&messageId=task%2Fwith+space',
    );
  });
});
