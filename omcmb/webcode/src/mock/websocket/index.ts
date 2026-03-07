type EventCallback<T = unknown> = (data: T) => void;

export class MockWebSocket {
  protected listeners: Map<string, Set<EventCallback>> = new Map();
  protected connected = false;
  protected intervalIds: ReturnType<typeof setInterval>[] = [];

  on<T = unknown>(event: string, callback: EventCallback<T>): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback as EventCallback);
  }

  off<T = unknown>(event: string, callback: EventCallback<T>): void {
    this.listeners.get(event)?.delete(callback as EventCallback);
  }

  protected emit<T = unknown>(event: string, data: T): void {
    this.listeners.get(event)?.forEach((cb) => {
      try {
        cb(data);
      } catch (err) {
        console.error(`[MockWebSocket] Error in listener for event "${event}":`, err);
      }
    });
  }

  connect(): void {
    if (this.connected) return;
    this.connected = true;
    this.emit('connect', { timestamp: new Date().toISOString() });
    this.onConnect();
  }

  disconnect(): void {
    if (!this.connected) return;
    this.connected = false;
    this.intervalIds.forEach((id) => clearInterval(id));
    this.intervalIds = [];
    this.onDisconnect();
    this.emit('disconnect', { timestamp: new Date().toISOString() });
  }

  isConnected(): boolean {
    return this.connected;
  }

  protected onConnect(): void {
    // Override in subclasses
  }

  protected onDisconnect(): void {
    // Override in subclasses
  }

  protected startInterval(callback: () => void, minMs: number, maxMs: number): void {
    const schedule = () => {
      if (!this.connected) return;
      const ms = Math.floor(Math.random() * (maxMs - minMs + 1)) + minMs;
      const id = setTimeout(() => {
        if (!this.connected) return;
        callback();
        schedule();
      }, ms);
      // Store as interval-like id for cleanup tracking
      this.intervalIds.push(id as unknown as ReturnType<typeof setInterval>);
    };
    schedule();
  }

  destroy(): void {
    this.disconnect();
    this.listeners.clear();
  }
}
