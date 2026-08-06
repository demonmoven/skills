import { describe, it, expect } from 'vitest';
import { parseDuration } from './duration.js';

describe('parseDuration', () => {
  it('parses bare number as seconds', () => {
    expect(parseDuration('3600')).toBe(3600);
    expect(parseDuration('0')).toBe(0);
  });

  it('parses s / m / h / d units', () => {
    expect(parseDuration('30s')).toBe(30);
    expect(parseDuration('5m')).toBe(300);
    expect(parseDuration('1h')).toBe(3600);
    expect(parseDuration('24h')).toBe(86400);
    expect(parseDuration('7d')).toBe(7 * 86400);
  });

  it('is case-insensitive', () => {
    expect(parseDuration('24H')).toBe(86400);
    expect(parseDuration('7D')).toBe(604800);
  });

  it('tolerates whitespace between number and unit', () => {
    expect(parseDuration('24 h')).toBe(86400);
  });

  it('rejects empty input', () => {
    expect(() => parseDuration('')).toThrow(/Empty duration/);
    expect(() => parseDuration('   ')).toThrow(/Empty duration/);
  });

  it('rejects invalid format', () => {
    expect(() => parseDuration('abc')).toThrow(/Invalid duration/);
    expect(() => parseDuration('24x')).toThrow(/Invalid duration/);
    expect(() => parseDuration('-5h')).toThrow(/Invalid duration/);
  });
});
