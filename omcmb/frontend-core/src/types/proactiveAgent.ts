export interface ProactiveScenario {
  key: string;
  name: string;
  description: string;
  status: 'ACTIVE' | 'DISABLED';
  rolloutMode: 'SHADOW' | 'PERCENTAGE' | 'FULL';
  rolloutPercentage: number;
  version: number;
  eventTypes: string[];
  allowedOperations: string[];
  deliverySurfaces: string[];
  dedupeWindowSeconds: number;
  rateLimitPerHour: number;
  timeoutSeconds: number;
  lastRunAt?: string;
  stats?: Record<string, number>;
}

export interface ProactiveOverview {
  scenarios: ProactiveScenario[];
  packages: Array<{ key: string; version: string; digest: string; status: string; createdAt: string }>;
  runs: Array<{ id: string; scenarioKey: string; status: string; rolloutMode: string; createdAt: string; completedAt?: string; errorCode?: string }>;
  connectorHealth: { status: string; workerId?: string; handbookDigest?: string; lastHeartbeatAt?: string; queueDepth: number; message?: string };
  stats: Record<string, number>;
}
