import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';

interface Tab { key: string; label: string; closable?: boolean }
let currentTabState: {
  tabs: Tab[];
  activeTabKey: string;
  closeTab: ReturnType<typeof vi.fn>;
  setActiveKey: ReturnType<typeof vi.fn>;
};

vi.mock('@core/store/tabStore', () => ({
  useTabStore: () => currentTabState,
}));

beforeEach(() => {
  currentTabState = {
    tabs: [
      { key: 'tab1', label: 'A', closable: true },
      { key: 'tab2', label: 'B', closable: true },
      { key: 'tab3', label: 'C', closable: false },
    ],
    activeTabKey: 'tab1',
    closeTab: vi.fn(),
    setActiveKey: vi.fn(),
  };
});

describe('useKeyboardShortcuts', () => {
  it('Ctrl+W on a closable active tab closes it', async () => {
    const { useKeyboardShortcuts } = await import('../useKeyboardShortcuts');
    renderHook(() => useKeyboardShortcuts());

    act(() => {
      window.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'w', ctrlKey: true }),
      );
    });

    expect(currentTabState.closeTab).toHaveBeenCalledWith('tab1');
  });

  it('Ctrl+W on a non-closable active tab does nothing', async () => {
    currentTabState.activeTabKey = 'tab3'; // closable: false
    const { useKeyboardShortcuts } = await import('../useKeyboardShortcuts');
    renderHook(() => useKeyboardShortcuts());

    act(() => {
      window.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'w', ctrlKey: true }),
      );
    });

    expect(currentTabState.closeTab).not.toHaveBeenCalled();
  });

  it('Ctrl+Tab moves to next tab (wraps around)', async () => {
    currentTabState.activeTabKey = 'tab3'; // last
    const { useKeyboardShortcuts } = await import('../useKeyboardShortcuts');
    renderHook(() => useKeyboardShortcuts());

    act(() => {
      window.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true }),
      );
    });

    expect(currentTabState.setActiveKey).toHaveBeenCalledWith('tab1');
  });

  it('Ctrl+Shift+Tab moves to previous tab (wraps around)', async () => {
    currentTabState.activeTabKey = 'tab1'; // first
    const { useKeyboardShortcuts } = await import('../useKeyboardShortcuts');
    renderHook(() => useKeyboardShortcuts());

    act(() => {
      window.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, shiftKey: true }),
      );
    });

    expect(currentTabState.setActiveKey).toHaveBeenCalledWith('tab3');
  });

  it('non-Ctrl key combinations are ignored', async () => {
    const { useKeyboardShortcuts } = await import('../useKeyboardShortcuts');
    renderHook(() => useKeyboardShortcuts());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'w' }));
    });

    expect(currentTabState.closeTab).not.toHaveBeenCalled();
    expect(currentTabState.setActiveKey).not.toHaveBeenCalled();
  });

  it('Ctrl+Tab on empty tab list is a no-op', async () => {
    currentTabState.tabs = [];
    const { useKeyboardShortcuts } = await import('../useKeyboardShortcuts');
    renderHook(() => useKeyboardShortcuts());

    act(() => {
      window.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true }),
      );
    });

    expect(currentTabState.setActiveKey).not.toHaveBeenCalled();
  });
});
