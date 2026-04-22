import http from '../http';

// --- Backend response types ---

interface BackendTestCase {
  id: string;
  name: string;
  description: string;
  category: string;
  steps: BackendTestStep[];
}

interface BackendTestStep {
  order: number;
  description: string;
  action: string;
  params: Record<string, unknown>;
}

interface BackendTestResult {
  test_case_id: string;
  test_name: string;
  category: string;
  passed: boolean;
  duration_ms: number;
  details: string;
  error: string;
}

interface BackendRunTestsResponse {
  device_sn: string;
  total: number;
  passed: number;
  failed: number;
  results: BackendTestResult[];
}

interface BackendTestCasesResponse {
  [category: string]: BackendTestCase[];
}

// --- Frontend types ---

export interface TestCase {
  id: string;
  name: string;
  description: string;
  category: string;
  steps: TestStep[];
}

export interface TestStep {
  order: number;
  description: string;
  action: string;
  params: Record<string, unknown>;
}

export interface TestResult {
  testCaseId: string;
  testName: string;
  category: string;
  passed: boolean;
  durationMs: number;
  details: string;
  error: string;
}

export interface RunTestsResponse {
  deviceSn: string;
  total: number;
  passed: number;
  failed: number;
  results: TestResult[];
}

export interface RunTestsRequest {
  deviceSn: string;
  categories?: string[];
}

export type TestCategory = 'protocol' | 'datamodel' | 'rpc' | 'inform';

// --- Mapping functions ---

function mapBackendTestCase(tc: BackendTestCase): TestCase {
  return {
    id: tc.id,
    name: tc.name,
    description: tc.description,
    category: tc.category,
    steps: (tc.steps || []).map((s) => ({
      order: s.order,
      description: s.description,
      action: s.action,
      params: s.params || {},
    })),
  };
}

function mapBackendTestResult(r: BackendTestResult): TestResult {
  return {
    testCaseId: r.test_case_id,
    testName: r.test_name,
    category: r.category,
    passed: r.passed,
    durationMs: r.duration_ms,
    details: r.details || '',
    error: r.error || '',
  };
}

// --- Exported service ---

export const interopApi = {
  async getTestCases(): Promise<Record<string, TestCase[]>> {
    const { data } = await http.get<BackendTestCasesResponse>('/interop/test-cases');
    const result: Record<string, TestCase[]> = {};
    for (const [category, cases] of Object.entries(data)) {
      result[category] = (cases || []).map(mapBackendTestCase);
    }
    return result;
  },

  async runTests(req: RunTestsRequest): Promise<RunTestsResponse> {
    const body = {
      device_sn: req.deviceSn,
      categories: req.categories,
    };
    const { data } = await http.post<BackendRunTestsResponse>('/interop/run', body);
    return {
      deviceSn: data.device_sn,
      total: data.total,
      passed: data.passed,
      failed: data.failed,
      results: (data.results || []).map(mapBackendTestResult),
    };
  },

  async runByCategory(
    category: string,
    req: RunTestsRequest
  ): Promise<RunTestsResponse> {
    const body = {
      device_sn: req.deviceSn,
    };
    const { data } = await http.post<BackendRunTestsResponse>(
      `/interop/run/${category}`,
      body
    );
    return {
      deviceSn: data.device_sn,
      total: data.total,
      passed: data.passed,
      failed: data.failed,
      results: (data.results || []).map(mapBackendTestResult),
    };
  },

  async validateDevice(
    deviceId: string,
    carrier: string,
    tech: string
  ): Promise<unknown> {
    const { data } = await http.post(`/interop/validate/${deviceId}`, null, {
      params: { carrier, tech },
    });
    return data;
  },
};
