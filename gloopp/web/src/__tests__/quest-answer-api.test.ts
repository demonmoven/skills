import assert from 'node:assert/strict'
import { answerQuest } from '../api/quests'

type FetchCall = {
  path: string
  init: RequestInit
}

const calls: FetchCall[] = []
const session = new Map<string, string>()

;(globalThis as unknown as { window: { location: { href: string } } }).window = {
  location: { href: 'http://localhost:5173/quests/q-123?t=test-token' },
}

;(globalThis as unknown as { sessionStorage: Storage }).sessionStorage = {
  getItem: (key: string) => session.get(key) ?? null,
  setItem: (key: string, value: string) => { session.set(key, value) },
  removeItem: (key: string) => { session.delete(key) },
  clear: () => { session.clear() },
  key: (index: number) => Array.from(session.keys())[index] ?? null,
  get length() { return session.size },
} as Storage

;(globalThis as unknown as { fetch: typeof fetch }).fetch = async (path, init) => {
  calls.push({ path: String(path), init: init || {} })
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    json: async () => ({ ok: true, qid: 'q-123', answer: { answer_id: 'ans-1' } }),
  } as Response
}

const res = await answerQuest('q-123', {
  question_id: 'question-7',
  answer: '按推荐方案继续',
})

assert.equal(res.ok, true)
assert.equal(calls.length, 1)
assert.equal(calls[0].path, '/api/quests/q-123/answer')
assert.equal(calls[0].init.method, 'POST')
assert.equal((calls[0].init.headers as Record<string, string>).Authorization, 'Bearer test-token')
assert.deepEqual(JSON.parse(String(calls[0].init.body)), {
  question_id: 'question-7',
  answer: '按推荐方案继续',
  source: 'dashboard',
})

console.log('\nquest-answer-api tests: PASSED 1 / 1')
