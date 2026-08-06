export const logger = {
  info(msg: string): void {
    console.log(`[trace] ${msg}`);
  },
  warn(msg: string): void {
    console.warn(`[trace] WARN: ${msg}`);
  },
  error(msg: string): void {
    console.error(`[trace] ERROR: ${msg}`);
  },
  debug(msg: string): void {
    if (process.env['TRACE_DEBUG']) {
      console.debug(`[trace] DEBUG: ${msg}`);
    }
  },
};
