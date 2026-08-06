import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mkdir, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

const tempDir = join(tmpdir(), `trace-session-test-${Date.now()}-${Math.random().toString(36).slice(2)}`);

vi.mock('node:os', async () => {
  const actual = await vi.importActual('node:os');
  return { ...actual, homedir: () => tempDir };
});

const { saveSessionState, saveSessionStateIfAbsent, loadSessionState, cleanupSessionState } = await import('./state.js');
import type { SessionState } from './state.js';

describe('session state', () => {
  beforeEach(async () => {
    await mkdir(join(tempDir, '.trace', 'sessions'), { recursive: true });
  });

  afterEach(async () => {
    await rm(tempDir, { recursive: true, force: true });
  });

  const sampleState: SessionState = {
    sessionId: 'test-session-123',
    startCommit: 'abc123def456',
    cwd: '/Users/test/project',
    startedAt: '2026-04-02T12:00:00+08:00',
  };

  it('saves and loads session state', async () => {
    await saveSessionState(sampleState);
    const loaded = await loadSessionState('test-session-123');
    expect(loaded).toEqual(sampleState);
  });

  it('does not overwrite an existing session state when saving if absent', async () => {
    await saveSessionState(sampleState);

    const replacement: SessionState = {
      ...sampleState,
      startCommit: 'newer-commit-after-user-work',
      startedAt: '2026-04-02T12:30:00+08:00',
    };

    const saved = await saveSessionStateIfAbsent(replacement);
    const loaded = await loadSessionState('test-session-123');

    expect(saved).toBe(false);
    expect(loaded).toEqual(sampleState);
  });

  it('writes a new session state when saving if absent and no state exists', async () => {
    const saved = await saveSessionStateIfAbsent(sampleState);
    const loaded = await loadSessionState('test-session-123');

    expect(saved).toBe(true);
    expect(loaded).toEqual(sampleState);
  });

  it('returns null for non-existent session', async () => {
    const loaded = await loadSessionState('non-existent');
    expect(loaded).toBeNull();
  });

  it('cleans up session state', async () => {
    await saveSessionState(sampleState);
    await cleanupSessionState('test-session-123');
    const loaded = await loadSessionState('test-session-123');
    expect(loaded).toBeNull();
  });

  it('cleanup is idempotent', async () => {
    await expect(cleanupSessionState('non-existent')).resolves.toBeUndefined();
  });
});
