import { post } from './api-client';
import type { ManagerConfig } from '../types/api';

export async function registerBehavior(config: ManagerConfig): Promise<void> {
  return post<void>('/api/lights/behavior/register', config);
}

export async function turnLightsOn(): Promise<void> {
  return post<void>('/api/lights/on');
}

export async function turnLightsOff(): Promise<void> {
  return post<void>('/api/lights/off');
}
