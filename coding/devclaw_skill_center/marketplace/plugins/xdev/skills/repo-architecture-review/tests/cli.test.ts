// tests/cli.test.ts
import { describe, it, expect } from 'vitest';
import { execa } from 'execa';

describe('cli', () => {
  it('prints version', async () => {
    const { stdout, exitCode } = await execa('tsx', ['scripts/cli.ts', '--version']);
    expect(exitCode).toBe(0);
    expect(stdout).toMatch(/^repo-architecture-review \d+\.\d+\.\d+$/);
  });
});
