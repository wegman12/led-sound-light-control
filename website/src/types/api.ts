export type Color = 'red' | 'green' | 'blue' | 'white';

export type BehaviorType = 'breathing' | 'flashing' | 'fixed' | 'skipper';

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

export type BehaviorConfigData =
  | FixedConfig
  | BreathingConfig
  | FlashingConfig
  | SkipperConfig;

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
