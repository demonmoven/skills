import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { dirname, join } from 'node:path';
import YAML from 'yaml';

const BUNDLED_B64 = {
  ak: 'REFMSDNYQTFFRzc0WFRVNEg5QUY=',
  sk: 'TVU2T24vUnZIUGV1bnBSUEo0Z21wR082ZWxwTWR3MytMRkJkRWRSUA==',
  bucket: 'c3RvbmUtY29zdHVkaW8tYm9l',
  region: 'Y24tYmVpamluZw==',
  endpoint: 'dG9zLWNuLW5vcnRoLWJvZS5ieXRlZC5vcmc=',
};

const GLOBAL_CONFIG_PATH = join(homedir(), '.trace', 'config.yaml');

function decode(b64: string): string {
  if (b64.startsWith('__PLACEHOLDER_')) return '';
  try {
    return Buffer.from(b64, 'base64').toString('utf-8');
  } catch {
    return '';
  }
}

export interface BundledTosCreds {
  ak: string;
  sk: string;
  bucket: string;
  region: string;
  endpoint: string;
}

export function getBundledTosCreds(): BundledTosCreds {
  return {
    ak: decode(BUNDLED_B64.ak),
    sk: decode(BUNDLED_B64.sk),
    bucket: decode(BUNDLED_B64.bucket),
    region: decode(BUNDLED_B64.region),
    endpoint: decode(BUNDLED_B64.endpoint),
  };
}

export function hasBundledCreds(): boolean {
  const c = getBundledTosCreds();
  return Boolean(c.ak || c.sk || c.bucket || c.region || c.endpoint);
}

export function applyBundledCredsIfMissing(
  configPath: string = GLOBAL_CONFIG_PATH,
  credsOverride?: BundledTosCreds,
): 'skipped' | 'written' | 'unchanged' {
  const creds = credsOverride ?? getBundledTosCreds();
  const hasAny = Boolean(creds.ak || creds.sk || creds.bucket || creds.region || creds.endpoint);
  if (!hasAny) return 'skipped';

  let doc: YAML.Document;
  if (existsSync(configPath)) {
    const src = readFileSync(configPath, 'utf-8');
    doc = YAML.parseDocument(src);
    if (doc.errors.length > 0) {
      console.error(
        `[xdev] Warn: ${configPath} parse error (${doc.errors[0].message}); skipping bundled TOS creds injection.`,
      );
      return 'skipped';
    }
  } else {
    doc = new YAML.Document({});
  }

  let tosNode = doc.get('tos');
  if (!YAML.isMap(tosNode)) {
    tosNode = new YAML.YAMLMap();
    doc.set('tos', tosNode);
  }
  const tosMap = tosNode as YAML.YAMLMap;

  let changed = false;
  const setIfMissing = (key: string, value: string): void => {
    if (!value) return;
    const existing = tosMap.get(key);
    if (existing === undefined || existing === null || existing === '') {
      tosMap.set(key, value);
      changed = true;
    }
  };

  setIfMissing('accessKey', creds.ak);
  setIfMissing('secretKey', creds.sk);
  setIfMissing('bucket', creds.bucket);
  setIfMissing('region', creds.region);
  setIfMissing('endpoint', creds.endpoint);

  if (!changed) return 'unchanged';

  mkdirSync(dirname(configPath), { recursive: true, mode: 0o700 });
  writeFileSync(configPath, doc.toString(), { mode: 0o600 });
  console.log(`[xdev] Pre-seeded TOS credentials to ${configPath}`);
  return 'written';
}
