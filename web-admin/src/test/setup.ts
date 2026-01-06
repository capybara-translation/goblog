import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from '../mocks/server'
import '@testing-library/jest-dom'

// テスト開始前にMSWサーバーを起動
beforeAll(() => {
  server.listen({
    onUnhandledRequest: 'warn',
  })
})

// 各テスト後にハンドラーをリセット
afterEach(() => {
  server.resetHandlers()
})

// 全テスト終了後にMSWサーバーを停止
afterAll(() => {
  server.close()
})
