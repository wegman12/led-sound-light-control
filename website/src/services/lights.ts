import { get, post } from './api-client';
import type { AudioConfigResponse, AudioStatusResponse, ManagerConfig, SimulationRequest, SimulationResponse } from '../types/api';

export async function registerBehavior(config: ManagerConfig): Promise<void> {
  return post<void>('/api/lights/behavior/register', config);
}

export async function turnLightsOn(): Promise<void> {
  return post<void>('/api/lights/on');
}

export async function turnLightsOff(): Promise<void> {
  return post<void>('/api/lights/off');
}

export async function startAudioStream(): Promise<void> {
  return post<void>('/api/lights/audio/start');
}

export async function stopAudioStream(): Promise<void> {
  return post<void>('/api/lights/audio/stop');
}

export async function getAudioStatus(): Promise<AudioStatusResponse> {
  return get<AudioStatusResponse>('/api/lights/audio/status');
}

export async function getAudioConfig(): Promise<AudioConfigResponse> {
  return get<AudioConfigResponse>('/api/lights/audio/config');
}

export async function simulateLEDs(request: SimulationRequest): Promise<SimulationResponse> {
  return post<SimulationResponse>('/api/lights/simulate', request);
}
