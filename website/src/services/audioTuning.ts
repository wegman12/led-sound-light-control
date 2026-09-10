import { get, post, put } from './api-client';
import type { AudioTuningConfig } from '../types/api';

export async function getTuningConfig(): Promise<AudioTuningConfig> {
  return get<AudioTuningConfig>('/api/audio/tuning/config');
}

export async function updateTuningConfig(config: AudioTuningConfig): Promise<{ message: string; config: AudioTuningConfig }> {
  return put<{ message: string; config: AudioTuningConfig }>('/api/audio/tuning/config', config);
}

export async function saveTuningConfig(): Promise<{ message: string; config: AudioTuningConfig }> {
  return post<{ message: string; config: AudioTuningConfig }>('/api/audio/tuning/config/save');
}
