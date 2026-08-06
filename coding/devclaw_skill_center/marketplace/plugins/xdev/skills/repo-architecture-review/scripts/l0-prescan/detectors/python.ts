// scripts/l0-prescan/detectors/python.ts
import { readFile } from 'node:fs/promises';
import { join, relative, dirname } from 'node:path';
import { fileExists, walkRepo } from '../../util/fs.js';

export interface PythonDetection {
  hasPyProject: boolean;
  hasSetupCfg: boolean;
  hasRequirementsTxt: boolean;
  hasImportLinterConfig: boolean;
  topLevelPackages: string[];
  fileCount: number;
  locTotal: number;
}

export async function detectPython(root: string): Promise<PythonDetection> {
  const pyProjectPath = join(root, 'pyproject.toml');
  const setupCfgPath = join(root, 'setup.cfg');
  const reqPath = join(root, 'requirements.txt');
  const importLinterPath = join(root, '.importlinter');

  const hasPyProject = await fileExists(pyProjectPath);
  const hasSetupCfg = await fileExists(setupCfgPath);
  const hasRequirementsTxt = await fileExists(reqPath);

  let hasImportLinterConfig = await fileExists(importLinterPath);
  if (!hasImportLinterConfig && hasPyProject) {
    const content = await readFile(pyProjectPath, 'utf-8');
    hasImportLinterConfig = content.includes('[tool.importlinter]');
  }
  if (!hasImportLinterConfig && hasSetupCfg) {
    const content = await readFile(setupCfgPath, 'utf-8');
    hasImportLinterConfig = content.includes('[importlinter]');
  }

  if (!hasPyProject && !hasSetupCfg && !hasRequirementsTxt) {
    return { hasPyProject, hasSetupCfg, hasRequirementsTxt, hasImportLinterConfig, topLevelPackages: [], fileCount: 0, locTotal: 0 };
  }

  const allFiles = await walkRepo(root);
  const pyFiles = allFiles.filter((f) => f.endsWith('.py'));

  const packagePaths = new Set<string>();
  let locTotal = 0;
  for (const f of pyFiles) {
    const rel = relative(root, f);
    const dir = dirname(rel);
    if (dir !== '.' && !dir.startsWith('..')) packagePaths.add(dir.replace(/\//g, '.'));
    try {
      const content = await readFile(f, 'utf-8');
      locTotal += content.split('\n').length;
    } catch { /* ignore */ }
  }

  return {
    hasPyProject,
    hasSetupCfg,
    hasRequirementsTxt,
    hasImportLinterConfig,
    topLevelPackages: [...packagePaths].sort(),
    fileCount: pyFiles.length,
    locTotal,
  };
}
