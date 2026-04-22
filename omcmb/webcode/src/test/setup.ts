import '@testing-library/jest-dom'

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
