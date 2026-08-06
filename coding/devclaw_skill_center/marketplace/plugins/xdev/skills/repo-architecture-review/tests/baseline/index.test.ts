import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { loadBaseline, saveBaseline, diffBaseline } from '../../scripts/baseline/index.js';
import type { Finding } from '../../scripts/types.js';

describe('baseline', () => {
  let dir: string;
  beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'bl-')); });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  const f = (id: string, severity: Finding['severity'] = 'medium'): Finding => ({
    id, category: 'layering', title: 't', rootCause: 'r', impact: 'i', actions: [],
    confidence: 'high', confidenceScore: 0.9, severity,
    signals: [], locations: [{ file: 'a.py' }], violationIds: ['v1'],
  });

  it('save + load round-trips', async () => {
    await saveBaseline(dir, { createdAt: '2026-04-22', findings: [f('id1')] });
    const loaded = await loadBaseline(dir);
    expect(loaded?.findings).toHaveLength(1);
  });

  it('diffBaseline: reports new + worsened, not unchanged', async () => {
    await saveBaseline(dir, { createdAt: '2026-04-22', findings: [f('id1', 'medium'), f('id2', 'high')] });
    const current = [f('id1', 'high'), f('id2', 'high'), f('id3', 'medium')];   // id1 worsened, id3 new
    const r = await diffBaseline(dir, current);
    expect(r.newOrWorse.map((x) => x.id).sort()).toEqual(['id1', 'id3']);
    expect(r.unchanged.map((x) => x.id)).toEqual(['id2']);
  });
});
