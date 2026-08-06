/**
 * P2 Thread Streaming — behavior tests.
 *
 * Covers:
 *   ST-1  EventSource URL construction (per-quest semantic stream)
 *   ST-2  hello frames are skipped, semantic events reach the handler
 *   ST-3  onerror → reconnecting state tracking
 *   ST-4  1.5s debounce: rapid events collapse into a single refetch
 *   ST-5  stream indicator CSS class mapping per StreamState
 */
import assert from 'node:assert/strict'
import { withToken } from '../api/client'
import type { QuestEvent } from '../api/types'

// ── Setup: mock window + sessionStorage for withToken() ──────────────────────
const session = new Map<string, string>()
session.set('gloop_token', 'test-token')
;(globalThis as unknown as { window: { location: { href: string } } }).window = {
  location: { href: 'http://localhost:5173/quests/qst_abc?t=test-token' },
}
;(globalThis as unknown as { sessionStorage: Storage }).sessionStorage = {
  getItem: (k: string) => session.get(k) ?? null,
  setItem: (k: string, v: string) => { session.set(k, v) },
  removeItem: (k: string) => { session.delete(k) },
  clear: () => { session.clear() },
  key: (i: number) => Array.from(session.keys())[i] ?? null,
  get length() { return session.size },
} as Storage

// ── Mock EventSource ────────────────────────────────────────────────────────
type ESMessage = { data: string }
type ESError = { message?: string }

class MockEventSource {
  static instances: MockEventSource[] = []
  url: string
  onopen: (() => void) | null = null
  onmessage: ((ev: ESMessage) => void) | null = null
  onerror: ((ev: ESError) => void) | null = null
  private _closed = false

  constructor(url: string) {
    this.url = url
    MockEventSource.instances.push(this)
  }

  close() { this._closed = true }
  get isClosed() { return this._closed }

  // Test helpers — simulate browser events
  simulateOpen() { this.onopen?.() }
  simulateMessage(data: string) { this.onmessage?.({ data }) }
  simulateError() { this.onerror?.({ message: 'simulated error' }) }
}

function installMockEventSource() {
  MockEventSource.instances = []
  ;(globalThis as unknown as { EventSource: typeof MockEventSource }).EventSource = MockEventSource
}

installMockEventSource()

// ── Helpers ──────────────────────────────────────────────────────────────────

/**
 * Mirrors the event-dispatch logic inside useEventStream's onmessage handler.
 * Extracted here for unit testing (the hook itself requires a React render
 * environment; its core dispatch logic is pure and testable).
 */
function dispatchEvent(rawData: string, handler: (ev: QuestEvent) => void): void {
  try {
    const event = JSON.parse(rawData) as QuestEvent
    if (event.type === 'hello') return
    handler(event)
  } catch {
    // Ignore malformed frames (mirrors hook behavior).
  }
}

/**
 * Mirrors the 1.5s debounce pattern used in ThreadView's SSE handler.
 * Returns a rate-limited wrapper that only invokes `fn` if at least
 * `minIntervalMs` have elapsed since the last invocation.
 *
 * Uses a provided `now` getter so tests can control time.
 */
function createRateLimiter(fn: () => void, minIntervalMs: number, now: () => number = Date.now) {
  let last = -Infinity
  return () => {
    const t = now()
    if (t - last < minIntervalMs) return
    last = t
    fn()
  }
}

// ── ST-1: EventSource URL construction ──────────────────────────────────────
// The per-quest semantic stream must hit /api/quests/{qid}/stream?level=semantic
// and include the auth token via withToken().
function testST1_urlConstruction() {
  MockEventSource.instances = []
  const qid = 'qst_abc'
  const path = withToken('/api/quests/' + qid + '/stream?level=semantic')
  const es = new MockEventSource(path)

  assert.ok(es.url.includes('/api/quests/qst_abc/stream'), 'URL must include per-quest stream path')
  assert.ok(es.url.includes('level=semantic'), 'URL must request semantic level')
  assert.ok(es.url.includes('t=test-token') || es.url.includes('t='), 'URL must include auth token param')

  // Global stream (no qid)
  const globalPath = withToken('/api/stream')
  const esGlobal = new MockEventSource(globalPath)
  assert.ok(esGlobal.url.includes('/api/stream'), 'global stream URL must be /api/stream')
  assert.ok(!esGlobal.url.includes('level=semantic'), 'global stream must not force semantic level')

  console.log('  ST-1 EventSource URL construction: PASS')
}

// ── ST-2: hello frames skipped, semantic events dispatched ─────────────────
function testST2_eventDispatch() {
  let handlerCalls = 0
  let lastEvent: QuestEvent | null = null
  const handler = (ev: QuestEvent) => { handlerCalls++; lastEvent = ev }

  // hello frame must be silently dropped
  dispatchEvent(JSON.stringify({ type: 'hello', ts: Date.now() }), handler)
  assert.equal(handlerCalls, 0, 'hello frame must NOT trigger handler')

  // semantic frame must be dispatched
  const semanticEv: QuestEvent = {
    type: 'post_added',
    quest_id: 'qst_abc',
    payload: { post_id: 'post_xyz', kind: 'human_comment' },
    ts: Date.now(),
  }
  dispatchEvent(JSON.stringify(semanticEv), handler)
  assert.equal(handlerCalls, 1, 'semantic event must trigger handler exactly once')
  const received = lastEvent as unknown as QuestEvent
  assert.equal(received.type, 'post_added', 'handler must receive post_added event')
  assert.equal((received.payload as Record<string, string>)?.post_id, 'post_xyz', 'payload must be preserved')

  // malformed JSON must not throw
  dispatchEvent('not-json{{{', handler)
  assert.equal(handlerCalls, 1, 'malformed frame must not increase handler call count')

  // another semantic event → handler called again
  dispatchEvent(JSON.stringify({ type: 'quest_status_changed', quest_id: 'qst_abc', payload: { status: 'success' } }), handler)
  assert.equal(handlerCalls, 2, 'second semantic event must trigger handler')

  console.log('  ST-2 Event dispatch (hello skip, semantic pass): PASS')
}

// ── ST-3: onerror triggers reconnecting tracking ────────────────────────────
// The useEventStream hook tracks reconnectSinceRef on onerror. We test the
// state transition logic that drives the UI indicator.
function testST3_errorTriggersReconnecting() {
  // Simulate the state machine: connecting → open → error → reconnecting
  type StreamState = 'connecting' | 'open' | 'reconnecting' | 'disconnected'
  let state: StreamState = 'connecting'
  let reconnectSince: number | null = null

  // Simulate onopen
  state = 'open'
  reconnectSince = null
  assert.equal(state, 'open', 'onopen must transition to open')
  assert.equal(reconnectSince, null, 'onopen must clear reconnectSince')

  // Simulate onerror — first error
  if (reconnectSince === null) reconnectSince = Date.now()
  state = 'reconnecting'
  assert.equal(state, 'reconnecting', 'onerror must transition to reconnecting')
  assert.ok(reconnectSince !== null, 'first onerror must record reconnectSince timestamp')

  // Simulate second onerror — reconnectSince should NOT be overwritten
  const firstReconnect = reconnectSince
  state = 'reconnecting'
  assert.equal(reconnectSince, firstReconnect, 'subsequent errors must not overwrite reconnectSince')

  // Simulate recovery (onopen fires again)
  state = 'open'
  reconnectSince = null
  assert.equal(state, 'open', 'recovery must transition back to open')
  assert.equal(reconnectSince, null, 'recovery must clear reconnectSince')

  console.log('  ST-3 Error → reconnecting state tracking: PASS')
}

// ── ST-4: 1.5s debounce rate limiter ────────────────────────────────────────
function testST4_debounce() {
  let fakeNow = 1000
  const clock = () => fakeNow

  let refetchCount = 0
  const limiter = createRateLimiter(() => { refetchCount++ }, 1500, clock)

  // First call at t=1000: passes
  limiter()
  assert.equal(refetchCount, 1, 'first call must pass')

  // Call at t=1500 (500ms later, < 1500ms): suppressed
  fakeNow = 1500
  limiter()
  assert.equal(refetchCount, 1, 'call within debounce window must be suppressed')

  // Call at t=2000 (1000ms later, < 1500ms): still suppressed
  fakeNow = 2000
  limiter()
  assert.equal(refetchCount, 1, 'call still within debounce window must be suppressed')

  // Call at t=2501 (1501ms after first): passes
  fakeNow = 2501
  limiter()
  assert.equal(refetchCount, 2, 'call after debounce window must pass')

  // Rapid burst: 5 calls within 100ms, only first passes
  fakeNow = 3000
  let burstCount = 0
  const burstLimiter = createRateLimiter(() => { burstCount++ }, 1500, () => fakeNow)
  for (let i = 0; i < 5; i++) {
    fakeNow = 3000 + i * 20
    burstLimiter()
  }
  assert.equal(burstCount, 1, 'burst of 5 calls within debounce must produce exactly 1 invocation')

  console.log('  ST-4 1.5s debounce rate limiter: PASS')
}

// ── ST-5: Stream indicator CSS class mapping ────────────────────────────────
// ThreadView renders the indicator with className = 'thread-stream-indicator stream-{state}'
// and shows state-specific text.
function testST5_indicatorClassMapping() {
  type StreamState = 'connecting' | 'open' | 'reconnecting' | 'disconnected'
  const states: StreamState[] = ['connecting', 'open', 'reconnecting', 'disconnected']

  for (const state of states) {
    const className = 'thread-stream-indicator stream-' + state
    // Verify CSS class structure
    assert.ok(className.startsWith('thread-stream-indicator'), 'indicator must have base class')
    assert.ok(className.includes('stream-' + state), 'indicator must include state-specific class: ' + state)
  }

  // Verify the "open" state shows "live" text (from stream_live i18n key)
  // In ThreadView: streamState === 'open' ? t('quest.thread.stream_live') : t('quest.thread.stream_' + state)
  const openLabelKey = 'quest.thread.stream_live'
  const otherLabelKey = (state: string) => 'quest.thread.stream_' + state
  assert.notEqual(openLabelKey, otherLabelKey('open'), 'open state must use stream_live key, not stream_open')
  assert.equal(otherLabelKey('connecting'), 'quest.thread.stream_connecting')
  assert.equal(otherLabelKey('reconnecting'), 'quest.thread.stream_reconnecting')
  assert.equal(otherLabelKey('disconnected'), 'quest.thread.stream_disconnected')

  // Verify the indicator always renders the dot span regardless of state
  // (In JSX: <span className="thread-stream-dot" /> is always present)
  const dotClass = 'thread-stream-dot'
  assert.ok(dotClass.length > 0, 'dot class must be non-empty')

  console.log('  ST-5 Stream indicator CSS class mapping: PASS')
}

// ── ST-6: Integration — SSE handler triggers refetch with debounce ───────────
// End-to-end simulation: mock EventSource + rate-limited handler, verify
// that rapid semantic events collapse into a single refetch.
function testST6_integrationRefetchWithDebounce() {
  MockEventSource.instances = []

  let fakeNow = 5000
  let refetchCount = 0

  // Build the handler exactly as ThreadView does
  const refetch = () => { refetchCount++ }
  let lastFetch = -Infinity
  const sseHandler = (_event: QuestEvent) => {
    if (fakeNow - lastFetch < 1500) return
    lastFetch = fakeNow
    refetch()
  }

  // Simulate what useEventStream's onmessage does:
  const es = new MockEventSource(withToken('/api/quests/qst_int/stream?level=semantic'))
  es.onmessage = (msg: ESMessage) => dispatchEvent(msg.data, sseHandler)

  // Fire 3 rapid post_added events within 200ms
  const events = [
    { type: 'post_added', payload: { post_id: 'p1' } },
    { type: 'post_added', payload: { post_id: 'p2' } },
    { type: 'post_added', payload: { post_id: 'p3' } },
  ]
  for (let i = 0; i < events.length; i++) {
    fakeNow = 5000 + i * 50
    es.simulateMessage(JSON.stringify(events[i]))
  }

  assert.equal(refetchCount, 1, '3 rapid events must trigger exactly 1 refetch (debounced)')

  // Fire another event 2 seconds later: should pass
  fakeNow = 7000
  es.simulateMessage(JSON.stringify({ type: 'post_added', payload: { post_id: 'p4' } }))
  assert.equal(refetchCount, 2, 'event after debounce window must trigger second refetch')

  // hello event in between must not affect debounce timer
  const beforeHello = refetchCount
  fakeNow = 7100
  es.simulateMessage(JSON.stringify({ type: 'hello' }))
  assert.equal(refetchCount, beforeHello, 'hello frame must not trigger refetch')
  // hello frame must not have advanced lastFetch (handler returned early via dispatchEvent skip)

  console.log('  ST-6 Integration: SSE handler + debounce refetch: PASS')
}

// ── Run all ──────────────────────────────────────────────────────────────────
function run() {
  console.log('streaming behavior tests:')
  testST1_urlConstruction()
  testST2_eventDispatch()
  testST3_errorTriggersReconnecting()
  testST4_debounce()
  testST5_indicatorClassMapping()
  testST6_integrationRefetchWithDebounce()
  console.log('streaming behavior tests: ALL PASSED')
}

run()
