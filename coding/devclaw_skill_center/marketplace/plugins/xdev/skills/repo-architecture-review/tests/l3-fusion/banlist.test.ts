import { describe, it, expect } from 'vitest';
import { containsBannedPhrase, hasEvidenceAnchor } from '../../scripts/l3-fusion/banlist.js';

describe('banlist', () => {
  it('catches filler phrases', () => {
    expect(containsBannedPhrase('Consider refactoring for maintainability')).toBe(true);
    expect(containsBannedPhrase('Improve modularity please')).toBe(true);
    expect(containsBannedPhrase('File src/domain/user.py:10 imports infra')).toBe(false);
  });

  it('requires evidence anchor like [file:line] or [metric=value]', () => {
    expect(hasEvidenceAnchor('The domain layer [src/domain/user.py:10] imports infra [metric=modularity=0.18]')).toBe(true);
    expect(hasEvidenceAnchor('The code is bad')).toBe(false);
  });
});
