import '@testing-library/jest-dom'

Object.defineProperty(globalThis, 'matchMedia', {
  configurable: true,
  value: () => ({ matches: false, addEventListener() {}, removeEventListener() {}, addListener() {}, removeListener() {}, dispatchEvent: () => false }),
})

if (!URL.createObjectURL) URL.createObjectURL = () => 'blob:test'
