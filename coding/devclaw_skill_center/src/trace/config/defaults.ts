import type { TraceConfig } from './schema.js';

export const defaultConfig: TraceConfig = {
  tos: {
    bucket: 'ai-coding-traces-archive',
    region: 'cn-beijing',
  },
  privacy: {
    collect_content: false,
    allowed_dirs: [],
  },
};
