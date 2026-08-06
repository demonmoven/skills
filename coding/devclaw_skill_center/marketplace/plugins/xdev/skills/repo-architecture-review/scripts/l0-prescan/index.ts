// scripts/l0-prescan/index.ts
import { resolve } from 'node:path';
import type { RepoProfile } from '../types.js';
import { detectGo } from './detectors/go.js';
import { detectPython } from './detectors/python.js';
import { detectDocker } from './detectors/docker.js';
import { detectGha } from './detectors/gha.js';
import { discoverLayering } from './layer-discovery.js';
import { walkRepo, fileExists } from '../util/fs.js';

export async function buildRepoProfile(rootPath: string, opts?: { skipLlm?: boolean }): Promise<RepoProfile> {
  const root = resolve(rootPath);
  const [go, py, dockerfiles, ghaWorkflows, allFiles] = await Promise.all([
    detectGo(root),
    detectPython(root),
    detectDocker(root),
    detectGha(root),
    walkRepo(root),
  ]);

  const languages: RepoProfile['languages'] = [];
  if (go.fileCount > 0) languages.push({ lang: 'go', fileCount: go.fileCount, locTotal: go.locTotal });
  if (py.fileCount > 0) languages.push({ lang: 'python', fileCount: py.fileCount, locTotal: py.locTotal });

  const detectedConfigs: RepoProfile['detectedConfigs'] = {};
  if (py.hasImportLinterConfig) detectedConfigs.importLinter = '.importlinter or pyproject.toml';
  if (await fileExists(`${root}/arch-go.yml`)) detectedConfigs.archGo = 'arch-go.yml';
  if (await fileExists(`${root}/.hadolint.yaml`)) detectedConfigs.hadolint = '.hadolint.yaml';
  if (await fileExists(`${root}/.actionlint.yaml`)) detectedConfigs.actionlint = '.actionlint.yaml';

  // Discover layering convention via LLM — runs in parallel with nothing else
  // since all detectors are already done. Adds ~5–15s but only runs once.
  const primaryLang = go.fileCount > 0 ? 'go' : py.fileCount > 0 ? 'python' : 'unknown';
  const layering = await discoverLayering({
    lang: primaryLang,
    modulePath: go.modulePath ?? '',
    topDirs: go.topLevelPackages.length > 0 ? go.topLevelPackages : py.topLevelPackages,
    packagePaths: go.topLevelPackages,
    skipLlm: opts?.skipLlm,
  });

  return {
    rootPath: root,
    languages,
    hasGoMod: go.hasGoMod,
    goModulePath: go.modulePath,
    hasPyProject: py.hasPyProject,
    hasSetupCfg: py.hasSetupCfg,
    hasRequirementsTxt: py.hasRequirementsTxt,
    dockerfiles,
    ghaWorkflows,
    topLevelPackages: {
      ...(go.topLevelPackages.length ? { go: go.topLevelPackages } : {}),
      ...(py.topLevelPackages.length ? { python: py.topLevelPackages } : {}),
    },
    workspaces: [],
    totalFiles: allFiles.length,
    totalLocMillions: (go.locTotal + py.locTotal) / 1_000_000,
    gitLogLineEstimate: 0,
    detectedConfigs,
    layering,
  };
}
