export type Color = 'red' | 'green' | 'blue' | 'white';

export type BehaviorType = 'breathing' | 'flashing' | 'fixed' | 'skipper' | 'joiner' | 'audio_modulator';

export type FrequencyBand = 'bass' | 'mid-low' | 'mid-high' | 'treble';

export interface FixedConfig {
  power_value: number;
}

export interface BreathingConfig {
  duration?: string;
  max_power_value?: number;
  min_power_value?: number;
}

export interface FlashingConfig {
  high_duration?: string;
  low_duration?: string;
  delay?: string;
}

export interface SkipperConfig {
  duration?: string;
  max_power_value?: number;
  min_power_value?: number;
}

export interface JoinedBehaviorItem {
  behavior_type: BehaviorType;
  duration: string;
  behavior: BehaviorConfigData;
}

export interface JoinerConfig {
  behaviors: JoinedBehaviorItem[];
}

export interface AudioModulatorConfig {
  frequency_band: FrequencyBand;
  min_power_value: number;
  max_power_value: number;
  max_magnitude: number;
  noise_threshold: number;
  smoothing: number;
  fallback_power: number | null;
}

export type BehaviorConfigData =
  | FixedConfig
  | BreathingConfig
  | FlashingConfig
  | SkipperConfig
  | JoinerConfig
  | AudioModulatorConfig;

export interface BehaviorConfig {
  behavior_type: BehaviorType;
  color: Color;
  config: BehaviorConfigData;
}

export interface ManagerConfig {
  behaviors: BehaviorConfig[];
}

export interface HealthResponse {
  status: string;
}

export interface AudioProfile {
  bass: number;
  mid_low: number;
  mid_high: number;
  treble: number;
}

export interface AudioStatusResponse {
  is_streaming: boolean;
  profile?: AudioProfile;
  message?: string;
}

export interface AudioBandConfig {
  max_magnitude: number;
  noise_threshold: number;
}

export interface AudioConfigResponse {
  bass: AudioBandConfig;
  mid_low: AudioBandConfig;
  mid_high: AudioBandConfig;
  treble: AudioBandConfig;
}

export interface SimulationResult {
  timestamp: number;
  red: number;
  green: number;
  blue: number;
  white: number;
}

export interface SimulationRequest {
  audio_csv_content: string;
  behavior_config: ManagerConfig;
}

export interface SimulationResponse {
  results: SimulationResult[];
  message?: string;
}

export interface AudioTuningBandConfig {
  max_magnitude: number;
  noise_threshold: number;
  min_power_value: number;
  max_power_value: number;
  smoothing: number;
}

export interface AudioTuningConfig {
  bass_cutoff: number;
  mid_high_cutoff: number;
  treble_cutoff: number;
  bass: AudioTuningBandConfig;
  mid_low: AudioTuningBandConfig;
  mid_high: AudioTuningBandConfig;
  treble: AudioTuningBandConfig;
}

export interface RawAudioData {
  bass: number;
  mid_low: number;
  mid_high: number;
  treble: number;
}

export interface ProcessedAudioData {
  bass: number;
  mid_low: number;
  mid_high: number;
  treble: number;
}

export interface AudioStreamData {
  timestamp: string;
  raw: RawAudioData;
  processed: ProcessedAudioData;
  config: AudioTuningConfig;
}

// Audio Configuration Management Types

export interface SavedAudioConfigSummary {
  name: string;
  display_name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface SavedAudioConfig extends SavedAudioConfigSummary {
  config: AudioTuningConfig;
}

export interface ConfigListResponse {
  configs: SavedAudioConfigSummary[];
  active_config: string;
}

export interface CreateConfigRequest {
  name: string;
  display_name: string;
  description: string;
  config: AudioTuningConfig;
}

export interface UpdateConfigRequest {
  display_name: string;
  description: string;
  config: AudioTuningConfig;
}

export interface SetActiveConfigRequest {
  config_name: string;
}

export interface DeleteConfigResponse {
  message: string;
  name: string;
}
