import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from '../mocks/server'
import '@testing-library/jest-dom'

// Start MSW server before tests
beforeAll(() => {
  server.listen({
    onUnhandledRequest: 'warn',
  })
})

// Reset handlers after each test
afterEach(() => {
  server.resetHandlers()
})

// Stop MSW server after all tests
afterAll(() => {
  server.close()
})
