import { describe, it, expect, vi, beforeEach } from 'vitest';

// Hoist mock so antd.message is a controllable spy used inside toast.ts.
vi.mock('antd', () => {
  const message = {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  };
  return { message };
});

import { message } from 'antd';
import { toast, withToast } from '../toast';

describe('toast', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('success calls message.success with given content', () => {
    toast.success('saved ok');
    expect(message.success).toHaveBeenCalledWith('saved ok');
  });

  it('warning calls message.warning', () => {
    toast.warning('careful');
    expect(message.warning).toHaveBeenCalledWith('careful');
  });

  it('info calls message.info', () => {
    toast.info('hello');
    expect(message.info).toHaveBeenCalledWith('hello');
  });

  it('error with Error object reads .message', () => {
    toast.error(new Error('boom'));
    expect(message.error).toHaveBeenCalledWith('boom');
  });

  it('error with Error object plus prefix joins both', () => {
    toast.error(new Error('boom'), 'Save failed');
    expect(message.error).toHaveBeenCalledWith('Save failed: boom');
  });

  it('error with string passes string through', () => {
    toast.error('plain reason');
    expect(message.error).toHaveBeenCalledWith('plain reason');
  });

  it('error with non-Error non-string falls back to "Unknown error"', () => {
    toast.error({ weird: true });
    expect(message.error).toHaveBeenCalledWith('Unknown error');
  });
});

describe('withToast', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('resolves with original value and fires success toast', async () => {
    const result = await withToast(Promise.resolve(42), { success: 'done' });
    expect(result).toBe(42);
    expect(message.success).toHaveBeenCalledWith('done');
    expect(message.error).not.toHaveBeenCalled();
  });

  it('rejects with original error and fires error toast', async () => {
    const err = new Error('bad');
    await expect(
      withToast(Promise.reject(err), { success: 'ok', errorPrefix: 'Save failed' }),
    ).rejects.toBe(err);
    expect(message.error).toHaveBeenCalledWith('Save failed: bad');
    expect(message.success).not.toHaveBeenCalled();
  });

  it('rejects without prefix uses raw error message', async () => {
    await expect(
      withToast(Promise.reject(new Error('xx')), { success: 'ok' }),
    ).rejects.toThrow('xx');
    expect(message.error).toHaveBeenCalledWith('xx');
  });
});
