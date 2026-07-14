import { displayRFStatusLabelOf, displayRFStatusOf, rfStatusLabelOf, rfStatusOf } from '../rfStatus';

describe('rfStatusOf', () => {
  it('normalizes MLN-style numeric values to switch states', () => {
    expect(rfStatusOf('1')).toBe('on');
    expect(rfStatusOf('0')).toBe('off');
    expect(rfStatusOf('3')).toBe('on');
    expect(rfStatusOf('2')).toBe('off');
  });

  it('supports boolean-like and textual RF states', () => {
    expect(rfStatusOf(' true ')).toBe('on');
    expect(rfStatusOf('OFF')).toBe('off');
    expect(rfStatusOf('enabled')).toBe('on');
    expect(rfStatusOf('disabled')).toBe('off');
  });

  it('preserves explicit abnormal states', () => {
    expect(rfStatusOf('error')).toBe('error');
    expect(rfStatusOf('failed')).toBe('error');
  });

  it('returns null for empty and unknown values', () => {
    expect(rfStatusOf(undefined)).toBeNull();
    expect(rfStatusOf('')).toBeNull();
    expect(rfStatusOf('unknown')).toBeNull();
    expect(rfStatusOf('4')).toBeNull();
  });
});

describe('rfStatusLabelOf', () => {
  const labels = { on: '射频开', off: '射频关', error: '异常' };

  it('maps numeric RF states to readable labels', () => {
    expect(rfStatusLabelOf('1', labels)).toBe('射频开');
    expect(rfStatusLabelOf('0', labels)).toBe('射频关');
    expect(rfStatusLabelOf('3', labels)).toBe('射频开');
    expect(rfStatusLabelOf('2', labels)).toBe('射频关');
  });

  it('uses the error label when provided', () => {
    expect(rfStatusLabelOf('error', labels)).toBe('异常');
  });

  it('forces RF OFF for offline devices even when the raw value is ON', () => {
    expect(displayRFStatusOf('1', false)).toBe('off');
    expect(displayRFStatusLabelOf('1', false, labels)).toBe('射频关');
  });
});
