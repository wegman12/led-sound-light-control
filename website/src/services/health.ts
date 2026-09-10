import { get } from './api-client';
import type { HealthResponse } from '../types/api';

export async function checkHealth(): Promise<HealthResponse> {
  return get<HealthResponse>('/health');
}
