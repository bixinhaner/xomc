import { formatProductName, normalTaskProtectionPresentation } from './shared';

const t = (key: string) => key;

describe('access-control display semantics', () => {
  it('uses an explicit label when the product name is absent', () => {
    expect(formatProductName(undefined, t)).toBe('deviceAccess.productNameUnknown');
    expect(formatProductName('   ', t)).toBe('deviceAccess.productNameUnknown');
    expect(formatProductName(' QRTB ', t)).toBe('QRTB');
  });

  it('describes the actual task protection state', () => {
    expect(normalTaskProtectionPresentation(true)).toEqual({
      color: 'red',
      messageKey: 'deviceAccess.frozen',
    });
    expect(normalTaskProtectionPresentation(false)).toEqual({
      color: 'green',
      messageKey: 'deviceAccess.allowed',
    });
  });
});
