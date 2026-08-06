import type { ToolAdapter } from './adapter.js';
import type { RepoProfile } from '../types.js';
import { importLinterAdapter } from './adapters/import-linter.js';
import { grimpAdapter } from './adapters/grimp.js';
import { pydepsAdapter } from './adapters/pydeps.js';
import { archGoAdapter } from './adapters/arch-go.js';
import { goArchAdapter } from './adapters/go-arch.js';
import { godaAdapter } from './adapters/goda.js';
import { gocycloAdapter } from './adapters/gocyclo.js';
import { goListAdapter } from './adapters/go-list.js';
import { hadolintAdapter } from './adapters/hadolint.js';
import { actionlintAdapter } from './adapters/actionlint.js';
import { codeMaatAdapter } from './adapters/code-maat.js';
import { sccAdapter } from './adapters/scc.js';

const REGISTERED: ToolAdapter[] = [
  importLinterAdapter,
  grimpAdapter,
  pydepsAdapter,
  archGoAdapter,
  goArchAdapter,
  godaAdapter,
  gocycloAdapter,
  goListAdapter,
  hadolintAdapter,
  actionlintAdapter,
  codeMaatAdapter,
  sccAdapter,
];

export function registerAdapter(a: ToolAdapter): void {
  REGISTERED.push(a);
}

export function pickAdaptersFor(profile: RepoProfile): ToolAdapter[] {
  return REGISTERED.filter((a) => a.detectApplies(profile));
}

export function allRegisteredAdapters(): ToolAdapter[] {
  return [...REGISTERED];
}
