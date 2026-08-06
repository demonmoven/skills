import express from 'express';
import { join } from 'node:path';
import { existsSync, readFileSync } from 'node:fs';
import { createApiRouter } from './routes.js';
import type { TraceConfig } from '../config/schema.js';
import { logger } from '../utils/logger.js';

declare const __dirname: string;

export interface ServerOptions {
  config: TraceConfig;
  port: number;
}

export interface Server {
  app: ReturnType<typeof express>;
  start: () => Promise<void>;
}

/**
 * Create and configure the Express server.
 * Returns the app and a start() function so the caller controls when to listen.
 */
export function createServer(options: ServerOptions): Server {
  const app = express();

  // JSON body parser
  app.use(express.json());

  // API routes
  app.use('/api', createApiRouter(options.config));

  // Serve frontend static files in production.
  // After esbuild bundle, __dirname is the package's dist/ directory,
  // and web/dist sits beside it under the package root.
  const webDistPath = join(__dirname, '..', 'web', 'dist');
  if (existsSync(webDistPath)) {
    app.use(express.static(webDistPath));

    // SPA fallback: all non-API, non-static-asset routes serve index.html
    // Read the file once at startup to avoid sendFile path issues with Express 5
    const indexHtml = readFileSync(join(webDistPath, 'index.html'), 'utf-8');
    app.use((_req, res, next) => {
      if (_req.path.startsWith('/api') || _req.path.startsWith('/assets')) {
        next();
        return;
      }
      res.type('html').send(indexHtml);
    });
  }

  function start(): Promise<void> {
    return new Promise((resolve) => {
      app.listen(options.port, () => {
        logger.info(`Dashboard running at http://localhost:${options.port}`);
        resolve();
      });
    });
  }

  return { app, start };
}
