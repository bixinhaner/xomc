export interface WaitForExpectedParameterValuesOptions {
  expected: ReadonlyMap<string, string>;
  read: () => Promise<ReadonlyMap<string, string>>;
  intervalMs?: number;
  timeoutMs?: number;
  signal?: AbortSignal;
}

export class ParameterReadbackTimeoutError extends Error {
  constructor() {
    super('parameter readback timed out before submitted values were observed');
    this.name = 'ParameterReadbackTimeoutError';
  }
}

function createAbortError(): Error {
  if (typeof DOMException !== 'undefined') {
    return new DOMException('The operation was aborted.', 'AbortError');
  }
  const error = new Error('The operation was aborted.');
  error.name = 'AbortError';
  return error;
}

function throwIfAborted(signal?: AbortSignal): void {
  if (signal?.aborted) {
    throw createAbortError();
  }
}

async function delayWithSignal(delayMs: number, signal?: AbortSignal): Promise<void> {
  if (!signal) {
    await new Promise<void>((resolve) => {
      globalThis.setTimeout(resolve, delayMs);
    });
    return;
  }

  await new Promise<void>((resolve, reject) => {
    const timerId = globalThis.setTimeout(() => {
      signal.removeEventListener('abort', onAbort);
      resolve();
    }, delayMs);
    const onAbort = () => {
      globalThis.clearTimeout(timerId);
      signal.removeEventListener('abort', onAbort);
      reject(createAbortError());
    };
    signal.addEventListener('abort', onAbort, { once: true });
  });
}

function parameterValuesMatch(
  expected: ReadonlyMap<string, string>,
  actual: ReadonlyMap<string, string>,
): boolean {
  for (const [path, expectedValue] of expected) {
    if (actual.get(path) !== expectedValue) {
      return false;
    }
  }
  return true;
}

export async function waitForExpectedParameterValues({
  expected,
  read,
  intervalMs = 500,
  timeoutMs = 30_000,
  signal,
}: WaitForExpectedParameterValuesOptions): Promise<ReadonlyMap<string, string>> {
  const deadlineAt = Date.now() + timeoutMs;
  while (true) {
    throwIfAborted(signal);
    const actual = await read();
    throwIfAborted(signal);
    if (parameterValuesMatch(expected, actual)) {
      return actual;
    }
    if (Date.now() >= deadlineAt) {
      throw new ParameterReadbackTimeoutError();
    }
    await delayWithSignal(intervalMs, signal);
  }
}
