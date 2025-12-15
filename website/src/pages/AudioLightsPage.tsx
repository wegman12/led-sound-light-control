import { useState, useEffect } from 'react';
import {
  Container,
  Typography,
  Box,
  Button,
  Backdrop,
  CircularProgress,
  Snackbar,
  Alert,
  Paper,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Slider,
  Switch,
  FormControlLabel,
  Chip,
  Divider,
  Stack,
  Tooltip,
  IconButton,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import MusicNoteIcon from '@mui/icons-material/MusicNote';
import {
  registerBehavior,
  turnLightsOn,
  startAudioStream,
  stopAudioStream,
  getAudioConfig,
  getAudioStatus,
  ApiError,
} from '../services';
import type {
  BehaviorConfig,
  ManagerConfig,
  Color,
  FrequencyBand,
  AudioConfigResponse,
  AudioStatusResponse,
  AudioModulatorConfig,
} from '../types/api';

interface ColorBandConfig {
  enabled: boolean;
  frequencyBand: FrequencyBand;
  minPower: number;
  maxPower: number;
  smoothing: number;
}

const FREQUENCY_BANDS: { value: FrequencyBand; label: string; description: string }[] = [
  { value: 'bass', label: 'Bass (0-150 Hz)', description: 'Kick drums, bass guitar, sub-bass' },
  { value: 'mid-low', label: 'Mid-Low (150-1000 Hz)', description: 'Rhythm guitars, lower vocals' },
  { value: 'mid-high', label: 'Mid-High (1000-2000 Hz)', description: 'Snare drums, lead vocals' },
  { value: 'treble', label: 'Treble (2000+ Hz)', description: 'Cymbals, hi-hats' },
];

const COLOR_INFO: Record<Color, { label: string; color: string }> = {
  red: { label: 'Red', color: '#f44336' },
  green: { label: 'Green', color: '#4caf50' },
  blue: { label: 'Blue', color: '#2196f3' },
  white: { label: 'White', color: '#ffffff' },
};

export default function AudioLightsPage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [audioStatus, setAudioStatus] = useState<AudioStatusResponse | null>(null);
  const [audioConfig, setAudioConfig] = useState<AudioConfigResponse | null>(null);
  const [statusLoading, setStatusLoading] = useState(true);

  // Configuration for each color
  const [redConfig, setRedConfig] = useState<ColorBandConfig>({
    enabled: true,
    frequencyBand: 'bass',
    minPower: 0.1,
    maxPower: 1.0,
    smoothing: 0.3,
  });

  const [greenConfig, setGreenConfig] = useState<ColorBandConfig>({
    enabled: true,
    frequencyBand: 'mid-low',
    minPower: 0.1,
    maxPower: 1.0,
    smoothing: 0.3,
  });

  const [blueConfig, setBlueConfig] = useState<ColorBandConfig>({
    enabled: true,
    frequencyBand: 'mid-high',
    minPower: 0.1,
    maxPower: 1.0,
    smoothing: 0.3,
  });

  const [whiteConfig, setWhiteConfig] = useState<ColorBandConfig>({
    enabled: true,
    frequencyBand: 'treble',
    minPower: 0.0,
    maxPower: 0.8,
    smoothing: 0.1,
  });

  // Load audio config and status on mount
  useEffect(() => {
    const loadAudioInfo = async () => {
      try {
        const [config, status] = await Promise.all([getAudioConfig(), getAudioStatus()]);
        setAudioConfig(config);
        setAudioStatus(status);
      } catch (err) {
        console.error('Failed to load audio info:', err);
      } finally {
        setStatusLoading(false);
      }
    };
    loadAudioInfo();
  }, []);

  const handleLoadPreset = (preset: 'full-rgb' | 'bass-only' | 'treble-sparkle') => {
    switch (preset) {
      case 'full-rgb':
        setRedConfig({ enabled: true, frequencyBand: 'bass', minPower: 0.1, maxPower: 1.0, smoothing: 0.3 });
        setGreenConfig({ enabled: true, frequencyBand: 'mid-low', minPower: 0.1, maxPower: 1.0, smoothing: 0.3 });
        setBlueConfig({ enabled: true, frequencyBand: 'mid-high', minPower: 0.1, maxPower: 1.0, smoothing: 0.3 });
        setWhiteConfig({ enabled: true, frequencyBand: 'treble', minPower: 0.0, maxPower: 0.8, smoothing: 0.1 });
        break;
      case 'bass-only':
        setRedConfig({ enabled: true, frequencyBand: 'bass', minPower: 0.1, maxPower: 1.0, smoothing: 0.3 });
        setGreenConfig({ enabled: false, frequencyBand: 'mid-low', minPower: 0.1, maxPower: 1.0, smoothing: 0.3 });
        setBlueConfig({ enabled: false, frequencyBand: 'mid-high', minPower: 0.1, maxPower: 1.0, smoothing: 0.3 });
        setWhiteConfig({ enabled: false, frequencyBand: 'treble', minPower: 0.0, maxPower: 0.8, smoothing: 0.1 });
        break;
      case 'treble-sparkle':
        setRedConfig({ enabled: true, frequencyBand: 'bass', minPower: 0.3, maxPower: 0.3, smoothing: 0.0 });
        setGreenConfig({ enabled: true, frequencyBand: 'mid-low', minPower: 0.2, maxPower: 0.2, smoothing: 0.0 });
        setBlueConfig({ enabled: true, frequencyBand: 'mid-high', minPower: 0.4, maxPower: 0.4, smoothing: 0.0 });
        setWhiteConfig({ enabled: true, frequencyBand: 'treble', minPower: 0.0, maxPower: 1.0, smoothing: 0.1 });
        break;
    }
  };

  const createAudioBehaviorConfig = (
    config: ColorBandConfig,
    audioConfig: AudioConfigResponse | null
  ): AudioModulatorConfig => {
    const bandKey = config.frequencyBand.replace('-', '_') as keyof AudioConfigResponse;
    const bandConfig = audioConfig?.[bandKey];

    return {
      frequency_band: config.frequencyBand,
      min_power_value: config.minPower,
      max_power_value: config.maxPower,
      scaling_factor: bandConfig?.scaling_factor || 0.000001,
      noise_threshold: bandConfig?.noise_threshold || 0,
      smoothing: config.smoothing,
      fallback_power: 0.0,
    };
  };

  const handleApplyConfiguration = async () => {
    setLoading(true);
    setError(null);

    try {
      // Build behavior config for enabled colors
      const behaviors: BehaviorConfig[] = [];
      const configs: [Color, ColorBandConfig][] = [
        ['red', redConfig],
        ['green', greenConfig],
        ['blue', blueConfig],
        ['white', whiteConfig],
      ];

      for (const [color, config] of configs) {
        if (config.enabled) {
          behaviors.push({
            behavior_type: 'audio_modulator',
            color,
            config: createAudioBehaviorConfig(config, audioConfig),
          });
        }
      }

      if (behaviors.length === 0) {
        throw new Error('At least one color must be enabled');
      }

      const managerConfig: ManagerConfig = { behaviors };

      // Start audio stream if not already streaming
      if (!audioStatus?.is_streaming) {
        await startAudioStream();
      }

      // Register the behavior
      await registerBehavior(managerConfig);

      // Turn on the lights
      await turnLightsOn();

      // Update status
      const newStatus = await getAudioStatus();
      setAudioStatus(newStatus);
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Failed to apply audio configuration';
      setError(message);
      setSnackbarOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const handleStopAudio = async () => {
    setLoading(true);
    setError(null);

    try {
      await stopAudioStream();
      const newStatus = await getAudioStatus();
      setAudioStatus(newStatus);
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Failed to stop audio stream';
      setError(message);
      setSnackbarOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const handleSnackbarClose = () => {
    setSnackbarOpen(false);
  };

  const renderColorConfig = (
    color: Color,
    config: ColorBandConfig,
    setConfig: React.Dispatch<React.SetStateAction<ColorBandConfig>>
  ) => {
    const colorInfo = COLOR_INFO[color];

    return (
      <Paper
        elevation={2}
        sx={{
          p: 3,
          opacity: config.enabled ? 1 : 0.5,
          transition: 'opacity 0.3s',
          border: config.enabled ? `2px solid ${colorInfo.color}` : '2px solid transparent',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Typography variant="h6" sx={{ color: colorInfo.color, fontWeight: 'bold' }}>
            {colorInfo.label}
          </Typography>
          <FormControlLabel
            control={
              <Switch checked={config.enabled} onChange={(e) => setConfig({ ...config, enabled: e.target.checked })} />
            }
            label="Enabled"
          />
        </Box>

        {config.enabled && (
          <>
            <FormControl fullWidth sx={{ mb: 2 }}>
              <InputLabel>Frequency Band</InputLabel>
              <Select
                value={config.frequencyBand}
                label="Frequency Band"
                onChange={(e) => setConfig({ ...config, frequencyBand: e.target.value as FrequencyBand })}
              >
                {FREQUENCY_BANDS.map((band) => (
                  <MenuItem key={band.value} value={band.value}>
                    <Box>
                      <Typography>{band.label}</Typography>
                      <Typography variant="caption" color="text.secondary">
                        {band.description}
                      </Typography>
                    </Box>
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <Box sx={{ mb: 2 }}>
              <Typography gutterBottom>
                Brightness Range: {(config.minPower * 100).toFixed(0)}% - {(config.maxPower * 100).toFixed(0)}%
              </Typography>
              <Slider
                value={[config.minPower, config.maxPower]}
                onChange={(_e, value) => {
                  const [min, max] = value as number[];
                  setConfig({ ...config, minPower: min, maxPower: max });
                }}
                min={0}
                max={1}
                step={0.05}
                valueLabelDisplay="auto"
                valueLabelFormat={(value) => `${(value * 100).toFixed(0)}%`}
              />
            </Box>

            <Box sx={{ mb: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Typography gutterBottom>Smoothing: {(config.smoothing * 100).toFixed(0)}%</Typography>
                <Tooltip title="Higher smoothing prevents flickering but slower response. Lower smoothing allows rapid changes.">
                  <IconButton size="small">
                    <InfoIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </Box>
              <Slider
                value={config.smoothing}
                onChange={(_e, value) => setConfig({ ...config, smoothing: value as number })}
                min={0}
                max={1}
                step={0.05}
                valueLabelDisplay="auto"
                valueLabelFormat={(value) => `${(value * 100).toFixed(0)}%`}
              />
            </Box>
          </>
        )}
      </Paper>
    );
  };

  return (
    <>
      <Container maxWidth="lg">
        <Box
          sx={{
            mt: 4,
            display: 'flex',
            flexDirection: 'column',
            gap: 3,
          }}
        >
          <Box sx={{ textAlign: 'center' }}>
            <Typography variant="h4" component="h1" gutterBottom sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 1 }}>
              <MusicNoteIcon fontSize="large" />
              Audio-Reactive Lighting
            </Typography>
            <Typography variant="body1" color="text.secondary">
              Configure LED colors to respond to different audio frequency bands
            </Typography>
          </Box>

          {/* Audio Status */}
          {!statusLoading && audioStatus && (
            <Paper elevation={1} sx={{ p: 2, backgroundColor: audioStatus.is_streaming ? '#e8f5e9' : '#fff3e0' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Box>
                  <Typography variant="subtitle1" fontWeight="bold">
                    Audio Status: {audioStatus.is_streaming ? 'Streaming' : 'Not Streaming'}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {audioStatus.message}
                  </Typography>
                </Box>
                {audioStatus.is_streaming && (
                  <Button variant="outlined" color="error" onClick={handleStopAudio} disabled={loading}>
                    Stop Audio Stream
                  </Button>
                )}
              </Box>
            </Paper>
          )}

          {/* Presets */}
          <Paper elevation={2} sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Quick Presets
            </Typography>
            <Stack direction="row" spacing={2} flexWrap="wrap" useFlexGap>
              <Button variant="outlined" onClick={() => handleLoadPreset('full-rgb')}>
                Full RGB Visualization
              </Button>
              <Button variant="outlined" onClick={() => handleLoadPreset('bass-only')}>
                Bass Only (Red)
              </Button>
              <Button variant="outlined" onClick={() => handleLoadPreset('treble-sparkle')}>
                Treble Sparkle (Ambient + White)
              </Button>
            </Stack>
          </Paper>

          <Divider>
            <Chip label="Custom Configuration" />
          </Divider>

          {/* Color Configuration */}
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 3 }}>
            {renderColorConfig('red', redConfig, setRedConfig)}
            {renderColorConfig('green', greenConfig, setGreenConfig)}
            {renderColorConfig('blue', blueConfig, setBlueConfig)}
            {renderColorConfig('white', whiteConfig, setWhiteConfig)}
          </Box>

          {/* Apply Button */}
          <Paper elevation={2} sx={{ p: 3, textAlign: 'center' }}>
            <Button
              variant="contained"
              size="large"
              onClick={handleApplyConfiguration}
              disabled={loading || (!redConfig.enabled && !greenConfig.enabled && !blueConfig.enabled && !whiteConfig.enabled)}
              fullWidth
              sx={{ maxWidth: 400 }}
            >
              Apply Configuration & Start Audio
            </Button>
            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
              This will start audio streaming and apply your configuration
            </Typography>
          </Paper>
        </Box>
      </Container>

      {/* Loading Overlay */}
      <Backdrop sx={{ color: '#fff', zIndex: (theme) => theme.zIndex.drawer + 1 }} open={loading}>
        <CircularProgress color="inherit" />
      </Backdrop>

      {/* Error Snackbar */}
      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={handleSnackbarClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={handleSnackbarClose} severity="error" sx={{ width: '100%' }}>
          {error}
        </Alert>
      </Snackbar>
    </>
  );
}
