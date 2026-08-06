export function parseDuration(input: string): number {
  const s = input.trim();
  if (s === '') throw new Error('Empty duration');
  const m = /^(\d+)\s*([smhd]?)$/i.exec(s);
  if (!m) throw new Error(`Invalid duration: "${input}" (expected e.g. "3600", "1h", "7d")`);
  const n = parseInt(m[1] ?? '', 10);
  const unit = (m[2] ?? '').toLowerCase();
  switch (unit) {
    case '':
    case 's':
      return n;
    case 'm':
      return n * 60;
    case 'h':
      return n * 3600;
    case 'd':
      return n * 86400;
    default:
      throw new Error(`Invalid duration unit: "${unit}"`);
  }
}
