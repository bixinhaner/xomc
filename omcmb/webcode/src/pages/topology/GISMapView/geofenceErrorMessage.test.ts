import type { IntlShape } from 'react-intl';
import { describe, expect, it } from 'vitest';
import { geofenceErrorMessage } from './geofenceErrorMessage';

const intl = {
  formatMessage: ({ id }: { id: string }) => `translated:${id}`,
} as IntlShape;

describe('geofenceErrorMessage', () => {
  it('maps a known business code without exposing the backend error chain', () => {
    const error = Object.assign(
      new Error('duplicate key value violates unique constraint'),
      { bizCode: 2602 },
    );

    expect(geofenceErrorMessage(intl, error)).toBe(
      'translated:geofence.validation.nameDuplicate',
    );
  });

  it('uses the safe generic message for unknown failures', () => {
    expect(geofenceErrorMessage(intl, new Error('internal SQL error'))).toBe(
      'translated:geofence.message.operationFailed',
    );
  });
});
