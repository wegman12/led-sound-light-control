import { get, post, put, del } from './api-client';
import type {
  ConfigListResponse,
  SavedAudioConfig,
  CreateConfigRequest,
  UpdateConfigRequest,
  SetActiveConfigRequest,
  DeleteConfigResponse,
} from '../types/api';

const AUDIO_CONFIGS_ENDPOINT = '/api/audio/configs';

export async function listConfigs(): Promise<ConfigListResponse> {
  return get<ConfigListResponse>(AUDIO_CONFIGS_ENDPOINT);
}

export async function getConfig(name: string): Promise<SavedAudioConfig> {
  return get<SavedAudioConfig>(`${AUDIO_CONFIGS_ENDPOINT}/${encodeURIComponent(name)}`);
}

export async function createConfig(request: CreateConfigRequest): Promise<SavedAudioConfig> {
  return post<SavedAudioConfig>(AUDIO_CONFIGS_ENDPOINT, request);
}

export async function updateConfig(name: string, request: UpdateConfigRequest): Promise<SavedAudioConfig> {
  return put<SavedAudioConfig>(`${AUDIO_CONFIGS_ENDPOINT}/${encodeURIComponent(name)}`, request);
}

export async function deleteConfig(name: string): Promise<DeleteConfigResponse> {
  return del<DeleteConfigResponse>(`${AUDIO_CONFIGS_ENDPOINT}/${encodeURIComponent(name)}`);
}

export async function getActiveConfig(): Promise<SavedAudioConfig> {
  return get<SavedAudioConfig>(`${AUDIO_CONFIGS_ENDPOINT}/active`);
}

export async function setActiveConfig(configName: string): Promise<SavedAudioConfig> {
  const request: SetActiveConfigRequest = { config_name: configName };
  return put<SavedAudioConfig>(`${AUDIO_CONFIGS_ENDPOINT}/active`, request);
}
