export interface WaitForExpectedParameterValuesOptions {
  expected: ReadonlyMap<string, string>;
  read: (signal: AbortSignal) => Promise<ReadonlyMap<string, string>>;
  intervalMs?: number;
  timeoutMs?: number;
  signal?: AbortSignal;
}

export interface WaitForReadbackOptions<T> {
  read: (signal: AbortSignal) => Promise<T>;
  matches: (actual: T) => boolean;
  intervalMs?: number;
  timeoutMs?: number;
  signal?: AbortSignal;
}

export interface SubmittedReadbackState {
  taskId?: string;
  syncedForTaskId?: string;
  submittedDraftRevision?: number;
  currentDraftRevision: number;
}

export function canApplySubmittedReadback({
  taskId,
  syncedForTaskId,
  submittedDraftRevision,
  currentDraftRevision,
}: SubmittedReadbackState): boolean {
  return Boolean(
    taskId
    && syncedForTaskId !== taskId
    && submittedDraftRevision !== undefined
    && submittedDraftRevision === currentDraftRevision,
  );
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
  throwIfAborted(signal);
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
    if (signal.aborted) {
      onAbort();
    }
  });
}

async function readWithDeadline<T>(
  read: (signal: AbortSignal) => Promise<T>,
  remainingMs: number,
  signal?: AbortSignal,
): Promise<T> {
  throwIfAborted(signal);
  const readController = new AbortController();

  return new Promise<T>((resolve, reject) => {
    let settled = false;
    const finish = (
      callback: (value: T | Error) => void,
      value: T | Error,
    ) => {
      if (settled) return;
      settled = true;
      globalThis.clearTimeout(timeoutId);
      signal?.removeEventListener('abort', onAbort);
      callback(value);
    };
    const onAbort = () => {
      readController.abort();
      finish((error) => reject(error), createAbortError());
    };
    const timeoutId = globalThis.setTimeout(() => {
      readController.abort();
      finish((error) => reject(error), new ParameterReadbackTimeoutError());
    }, Math.max(0, remainingMs));

    signal?.addEventListener('abort', onAbort, { once: true });
    if (signal?.aborted) {
      onAbort();
      return;
    }
    void Promise.resolve()
      .then(() => read(readController.signal))
      .then(
        (value) => finish((result) => resolve(result as T), value),
        (error: unknown) => finish(
          (reason) => reject(reason),
          error instanceof Error ? error : new Error(String(error)),
        ),
      );
  });
}

function booleanReadbackValue(value: string): boolean | undefined {
  const normalized = value.trim().toLowerCase();
  if (['1', 'true', 'on', 'enable', 'enabled'].includes(normalized)) return true;
  if (['0', 'false', 'off', 'disable', 'disabled'].includes(normalized)) return false;
  return undefined;
}

export function parameterReadbackValuesMatch(expected: string, actual: string): boolean {
  if (expected === actual) return true;
  const expectedBoolean = booleanReadbackValue(expected);
  const actualBoolean = booleanReadbackValue(actual);
  return expectedBoolean !== undefined
    && actualBoolean !== undefined
    && expectedBoolean === actualBoolean;
}

export function parameterReadbackMapsMatch(
  expected: ReadonlyMap<string, string>,
  actual: ReadonlyMap<string, string>,
): boolean {
  for (const [path, expectedValue] of expected) {
    const actualValue = actual.get(path);
    if (actualValue === undefined || !parameterReadbackValuesMatch(expectedValue, actualValue)) {
      return false;
    }
  }
  return true;
}

export async function waitForReadback<T>({
  read,
  matches,
  intervalMs = 500,
  timeoutMs = 30_000,
  signal,
}: WaitForReadbackOptions<T>): Promise<T> {
  const deadlineAt = Date.now() + timeoutMs;
  while (true) {
    throwIfAborted(signal);
    const remainingBeforeRead = deadlineAt - Date.now();
    if (remainingBeforeRead <= 0) {
      throw new ParameterReadbackTimeoutError();
    }
    const actual = await readWithDeadline(read, remainingBeforeRead, signal);
    throwIfAborted(signal);
    if (matches(actual)) {
      return actual;
    }
    const remainingBeforeDelay = deadlineAt - Date.now();
    if (remainingBeforeDelay <= 0) {
      throw new ParameterReadbackTimeoutError();
    }
    await delayWithSignal(Math.min(intervalMs, remainingBeforeDelay), signal);
  }
}

export async function waitForExpectedParameterValues({
  expected,
  read,
  intervalMs = 500,
  timeoutMs = 30_000,
  signal,
}: WaitForExpectedParameterValuesOptions): Promise<ReadonlyMap<string, string>> {
  return waitForReadback({
    read,
    matches: (actual) => parameterReadbackMapsMatch(expected, actual),
    intervalMs,
    timeoutMs,
    signal,
  });
}
