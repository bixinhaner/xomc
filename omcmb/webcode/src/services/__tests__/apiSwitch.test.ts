import { describe, it, expect, vi } from 'vitest'
import { createApiSwitch } from '@/services/apiSwitch'

describe('createApiSwitch', () => {
  it('returns real service when VITE_USE_MOCK is not "true"', () => {
    const mockService = { list: vi.fn() }
    const realService = { list: vi.fn() }

    // In test environment, import.meta.env.VITE_USE_MOCK is undefined,
    // so useMock is false and createApiSwitch should return realService.
    const result = createApiSwitch(mockService, realService)
    expect(result).toBe(realService)
  })

  it('returns a service object with the expected shape', () => {
    const mockService = {
      getList: vi.fn(),
      getById: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
    }
    const realService = {
      getList: vi.fn(),
      getById: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
    }

    const result = createApiSwitch(mockService, realService)
    expect(result).toHaveProperty('getList')
    expect(result).toHaveProperty('getById')
    expect(result).toHaveProperty('create')
    expect(result).toHaveProperty('update')
    expect(result).toHaveProperty('remove')
  })
})
