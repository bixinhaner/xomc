import '@testing-library/jest-dom'

// antd 在 jsdom 环境下依赖 window.matchMedia（Grid/Steps 等组件 useBreakpoint）。
// 提供 stub；具体响应式测试（useResponsive.test.tsx）自己 Object.defineProperty 覆盖。
Object.defineProperty(globalThis, 'matchMedia', {
  writable: true,
  configurable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  }),
})

// jsdom 自带 localStorage，但在 vitest 的 jsdom 环境里 zustand persist middleware
// 偶发读到 storage.setItem is not a function。统一用稳定的内存实现覆盖，
// 保证测试隔离、可预测。
{
  const store = new Map<string, string>()
  const memoryStorage: Storage = {
    get length() {
      return store.size
    },
    clear: () => store.clear(),
    getItem: (key: string) => (store.has(key) ? store.get(key)! : null),
    key: (index: number) => Array.from(store.keys())[index] ?? null,
    removeItem: (key: string) => {
      store.delete(key)
    },
    setItem: (key: string, value: string) => {
      store.set(key, String(value))
    },
  }

  Object.defineProperty(globalThis, 'localStorage', {
    value: memoryStorage,
    writable: false,
    configurable: true,
  })
  Object.defineProperty(globalThis, 'sessionStorage', {
    value: memoryStorage,
    writable: false,
    configurable: true,
  })
}

// antd 6 的 TextArea/自适应组件经 @rc-component/resize-observer 使用 ResizeObserver,
// jsdom 未实现,提供 no-op stub(仅测试环境,浏览器原生支持)。
if (typeof globalThis.ResizeObserver === 'undefined') {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  globalThis.ResizeObserver = ResizeObserverStub as unknown as typeof ResizeObserver
}
