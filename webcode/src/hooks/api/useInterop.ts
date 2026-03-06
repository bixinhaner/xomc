import { useQuery, useMutation } from '@tanstack/react-query';
import {
  interopApi,
  type RunTestsRequest,
} from '@/services/api/interopApi';

export function useTestCases() {
  return useQuery({
    queryKey: ['interop', 'test-cases'],
    queryFn: () => interopApi.getTestCases(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useRunTests() {
  return useMutation({
    mutationFn: (req: RunTestsRequest) => interopApi.runTests(req),
  });
}

export function useRunTestsByCategory() {
  return useMutation({
    mutationFn: ({ category, req }: { category: string; req: RunTestsRequest }) =>
      interopApi.runByCategory(category, req),
  });
}

export function useValidateDevice() {
  return useMutation({
    mutationFn: ({
      deviceId,
      carrier,
      tech,
    }: {
      deviceId: string;
      carrier: string;
      tech: string;
    }) => interopApi.validateDevice(deviceId, carrier, tech),
  });
}
