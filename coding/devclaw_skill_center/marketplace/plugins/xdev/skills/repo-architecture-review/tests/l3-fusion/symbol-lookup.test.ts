import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fetchSymbols } from '../../scripts/l3-fusion/symbol-lookup.js';

describe('fetchSymbols', () => {
  let dir: string;
  beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'sl-')); });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('returns enclosing Python function body', async () => {
    const file = join(dir, 'user.py');
    await writeFile(file, [
      'def before():',
      '    return 0',
      '',
      'def load(u):',
      '    # some comment',
      '    from infra.db import get',
      '    return get(u)',
      '',
      'def after():',
      '    return 1',
      '',
    ].join('\n'));
    const got = await fetchSymbols({
      repoRoot: dir,
      allowedLocations: [{ file: 'user.py', line: 6 }],
      requests: [{ file: 'user.py', line: 6 }],
      maxLinesPerSymbol: 150,
      maxSymbols: 3,
    });
    expect(got).toHaveLength(1);
    expect(got[0].source).toContain('def load(u):');
    expect(got[0].source).toContain('return get(u)');
    expect(got[0].source).not.toContain('def before');
    expect(got[0].source).not.toContain('def after');
  });

  it('rejects requests outside allowed locations', async () => {
    const file = join(dir, 'user.py');
    await writeFile(file, 'def load(u):\n    return 1\n');
    const got = await fetchSymbols({
      repoRoot: dir,
      allowedLocations: [{ file: 'user.py', line: 1 }],
      requests: [{ file: 'secret.py', line: 1 }],    // not in allowed
      maxLinesPerSymbol: 150,
      maxSymbols: 3,
    });
    expect(got).toHaveLength(0);
  });

  it('caps at maxSymbols and truncates oversize bodies', async () => {
    const file = join(dir, 'big.py');
    const big = ['def huge():', ...Array(200).fill('    pass')].join('\n');
    await writeFile(file, big);
    const got = await fetchSymbols({
      repoRoot: dir,
      allowedLocations: [{ file: 'big.py', line: 1 }],
      requests: [{ file: 'big.py', line: 1 }, { file: 'big.py', line: 2 }, { file: 'big.py', line: 3 }, { file: 'big.py', line: 4 }],
      maxLinesPerSymbol: 50,
      maxSymbols: 3,
    });
    expect(got.length).toBeLessThanOrEqual(3);
    for (const s of got) {
      expect(s.source.split('\n').length).toBeLessThanOrEqual(51);       // 50 + truncation marker
      expect(s.truncated).toBe(true);
    }
  });
});
