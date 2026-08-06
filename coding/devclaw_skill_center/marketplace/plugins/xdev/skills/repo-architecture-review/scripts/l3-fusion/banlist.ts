const BANNED = [
  /consider refactoring/i,
  /improve maintainability/i,
  /improve modularity/i,
  /enhance readability/i,
  /\bmay want to\b/i,
  /\bshould probably\b/i,
  /\boverall\s+quality\b/i,
  /best practice[s]?\b/i,
];

export function containsBannedPhrase(text: string): boolean {
  return BANNED.some((re) => re.test(text));
}

export function hasEvidenceAnchor(text: string): boolean {
  // Either [file:line] or [metric=...] or [co-change=...] style anchors.
  return /\[[\w./-]+(?::\d+)?\]|\[[^=\]]+=[^\]]+\]/.test(text);
}
