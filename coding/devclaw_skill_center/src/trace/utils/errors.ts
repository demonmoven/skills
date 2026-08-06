export class TraceError extends Error {
  constructor(
    message: string,
    public readonly code: string,
  ) {
    super(message);
    this.name = 'TraceError';
  }
}

export class AuthRequiredError extends TraceError {
  constructor() {
    super(
      'Not authenticated. Run `xdev trace auth login` first.',
      'AUTH_REQUIRED',
    );
    this.name = 'AuthRequiredError';
  }
}

export class FileNotFoundError extends TraceError {
  constructor(filePath: string) {
    super(`File not found: ${filePath}`, 'FILE_NOT_FOUND');
    this.name = 'FileNotFoundError';
  }
}

export class AdapterDetectionError extends TraceError {
  constructor(filePath: string) {
    super(
      `Cannot detect tool type for: ${filePath}. Use --source to specify.`,
      'ADAPTER_DETECTION_FAILED',
    );
    this.name = 'AdapterDetectionError';
  }
}
