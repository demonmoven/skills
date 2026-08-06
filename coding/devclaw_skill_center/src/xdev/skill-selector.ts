import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { createInterface } from 'node:readline/promises';
import { PLUGIN_SRC_DIR } from './paths.js';
import { parseIndexList } from './marketplace/prompt-selection.js';

export interface SkillInfo {
  name: string;
  description: string;
}

const MAX_DESC_LEN = 60;

function extractDescription(content: string): string {
  const fmMatch = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!fmMatch) return '';
  const match = fmMatch[1].match(/^description:\s*["']?(.*?)["']?\s*$/m);
  const raw = match?.[1] ?? '';
  if (raw.length <= MAX_DESC_LEN) return raw;
  return raw.slice(0, MAX_DESC_LEN - 1) + '…';
}

export function listAvailableSkills(): SkillInfo[] {
  const skillsDir = join(PLUGIN_SRC_DIR, 'skills');
  return readdirSync(skillsDir, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => {
      const mdPath = join(skillsDir, e.name, 'SKILL.md');
      let description = '';
      try {
        description = extractDescription(readFileSync(mdPath, 'utf-8'));
      } catch {
        /* no SKILL.md — show empty description */
      }
      return { name: e.name, description };
    })
    .sort((a, b) => a.name.localeCompare(b.name));
}

export async function promptSkillSelection(skills: SkillInfo[]): Promise<string[]> {
  if (skills.length === 0) return [];
  if (!process.stdin.isTTY) return skills.map((s) => s.name);

  console.log('');
  console.log(`[xdev] Available skills (${skills.length} total):`);
  const idxWidth = String(skills.length).length;
  const nameWidth = Math.max(...skills.map((s) => s.name.length));
  for (let i = 0; i < skills.length; i++) {
    const idx = String(i + 1).padStart(idxWidth);
    const name = skills[i].name.padEnd(nameWidth);
    const desc = skills[i].description ? ` — ${skills[i].description}` : '';
    console.log(`  ${idx}) ${name}${desc}`);
  }
  console.log('');
  console.log('Select skills to install (comma-separated, a=all, n=none):');

  const rl = createInterface({ input: process.stdin, output: process.stdout });
  let raw: string;
  try {
    raw = (await rl.question('> ')).trim().toLowerCase();
  } finally {
    rl.close();
  }

  if (raw === '' || raw === 'a') return skills.map((s) => s.name);
  if (raw === 'n' || raw === 'q') return [];

  const indices = parseIndexList(raw, skills.length);
  if (indices.length === 0) return skills.map((s) => s.name);
  return indices.map((i) => skills[i].name);
}
