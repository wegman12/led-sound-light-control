import { useState, useEffect, useCallback } from 'react';
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
  Slider,
  Stack,
  Card,
  CardContent,
  LinearProgress,
  Chip,
  FormControlLabel,
  Switch,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import TimelineIcon from '@mui/icons-material/Timeline';
import { simulateLEDs, getAudioConfig, ApiError } from '../services';
import type {
  ManagerConfig,
  SimulationResult,
  AudioConfigResponse,
  Color,
  FrequencyBand,
  AudioModulatorConfig,
} from '../types/api';

interface ColorBandConfig {
  enabled: boolean;
  frequencyBand: FrequencyBand;
  minPower: number;
  maxPower: number;
  smoothing: number;
}

const FREQUENCY_BANDS: { value: FrequencyBand; label: string }[] = [
  { value: 'bass', label: 'Bass (0-150 Hz)' },
  { value: 'mid-low', label: 'Mid-Low (150-1000 Hz)' },
  { value: 'mid-high', label: 'Mid-High (1000-2000 Hz)' },
  { value: 'treble', label: 'Treble (2000+ Hz)' },
];

const COLOR_INFO: Record<Color, { label: string; color: string }> = {
  red: { label: 'Red', color: '#f44336' },
  green: { label: 'Green', color: '#4caf50' },
  blue: { label: 'Blue', color: '#2196f3' },
  white: { label: 'White', color: '#ffffff' },
};

export default function AudioVisualizerPage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [csvFile, setCsvFile] = useState<File | null>(null);
  const [csvContent, setCsvContent] = useState<string>('');
  const [simulationResults, setSimulationResults] = useState<SimulationResult[] | null>(null);
  const [currentTimeIndex, setCurrentTimeIndex] = useState(0);
  const [audioConfig, setAudioConfig] = useState<AudioConfigResponse | null>(null);

  // Configuration for each color (preset: full RGB)
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

  // Load audio config on mount
  useEffect(() => {
    const loadAudioConfig = async () => {
      try {
        const config = await getAudioConfig();
        setAudioConfig(config);
      } catch (err) {
        console.error('Failed to load audio config:', err);
      }
    };
    loadAudioConfig();
  }, []);

  const handleFileUpload = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setCsvFile(file);

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      setCsvContent(content);
    };
    reader.readAsText(file);
  }, []);

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

  const handleRunSimulation = async () => {
    if (!csvContent) {
      setError('Please upload a CSV file first');
      setSnackbarOpen(true);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      // Build behavior config
      const behaviors = [];
      const configs: [Color, ColorBandConfig][] = [
        ['red', redConfig],
        ['green', greenConfig],
        ['blue', blueConfig],
        ['white', whiteConfig],
      ];

      for (const [color, config] of configs) {
        if (config.enabled) {
          behaviors.push({
            behavior_type: 'audio_modulator' as const,
            color,
            config: createAudioBehaviorConfig(config, audioConfig),
          });
        }
      }

      if (behaviors.length === 0) {
        throw new Error('At least one color must be enabled');
      }

      const managerConfig: ManagerConfig = { behaviors };

      // Run simulation
      const response = await simulateLEDs({
        audio_csv_content: csvContent,
        behavior_config: managerConfig,
      });

      setSimulationResults(response.results);
      setCurrentTimeIndex(0);
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Simulation failed';
      setError(message);
      setSnackbarOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const handleTimeChange = (_event: Event, value: number | number[]) => {
    setCurrentTimeIndex(value as number);
  };

  const currentResult = simulationResults?.[currentTimeIndex];
  const maxTimeIndex = (simulationResults?.length || 1) - 1;

  const renderColorBar = (color: Color, power: number) => {
    const colorInfo = COLOR_INFO[color];
    const percentage = power * 100;

    return (
      <Box key={color} sx={{ mb: 2 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
          <Typography variant="body2" sx={{ color: colorInfo.color, fontWeight: 'bold' }}>
            {colorInfo.label}
          </Typography>
          <Typography variant="body2">{percentage.toFixed(1)}%</Typography>
        </Box>
        <LinearProgress
          variant="determinate"
          value={percentage}
          sx={{
            height: 24,
            borderRadius: 1,
            backgroundColor: '#e0e0e0',
            '& .MuiLinearProgress-bar': {
              backgroundColor: colorInfo.color,
              borderRadius: 1,
            },
          }}
        />
      </Box>
    );
  };

  const renderColorConfig = (
    color: Color,
    config: ColorBandConfig,
    setConfig: React.Dispatch<React.SetStateAction<ColorBandConfig>>
  ) => {
    const colorInfo = COLOR_INFO[color];

    return (
      <Box key={color} sx={{ mb: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
          <FormControlLabel
            control={<Switch checked={config.enabled} onChange={(e) => setConfig({ ...config, enabled: e.target.checked })} size="small" />}
            label={<Typography sx={{ color: colorInfo.color, fontWeight: 'bold' }}>{colorInfo.label}</Typography>}
          />
        </Box>
        {config.enabled && (
          <>
            <FormControl fullWidth size="small" sx={{ mb: 1 }}>
              <InputLabel>Band</InputLabel>
              <Select
                value={config.frequencyBand}
                label="Band"
                onChange={(e) => setConfig({ ...config, frequencyBand: e.target.value as FrequencyBand })}
              >
                {FREQUENCY_BANDS.map((band) => (
                  <MenuItem key={band.value} value={band.value}>
                    {band.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <Typography variant="caption" gutterBottom>
              Range: {(config.minPower * 100).toFixed(0)}%-{(config.maxPower * 100).toFixed(0)}%, Smoothing: {(config.smoothing * 100).toFixed(0)}%
            </Typography>
          </>
        )}
      </Box>
    );
  };

  return (
    <>
      <Container maxWidth="lg">
        <Box sx={{ mt: 4, display: 'flex', flexDirection: 'column', gap: 3 }}>
          <Box sx={{ textAlign: 'center' }}>
            <Typography variant="h4" component="h1" gutterBottom sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 1 }}>
              <TimelineIcon fontSize="large" />
              Audio LED Visualizer
            </Typography>
            <Typography variant="body1" color="text.secondary">
              Upload audio CSV data and visualize LED behavior over time
            </Typography>
          </Box>

          <Paper elevation={2} sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              1. Upload Audio CSV File
            </Typography>
            <Stack direction="row" spacing={2} alignItems="center">
              <Button variant="contained" component="label" startIcon={<CloudUploadIcon />}>
                Choose File
                <input type="file" hidden accept=".csv" onChange={handleFileUpload} />
              </Button>
              {csvFile && (
                <Chip label={csvFile.name} color="success" onDelete={() => { setCsvFile(null); setCsvContent(''); }} />
              )}
            </Stack>
          </Paper>

          <Paper elevation={2} sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              2. Configure LED Behaviors
            </Typography>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2 }}>
              {renderColorConfig('red', redConfig, setRedConfig)}
              {renderColorConfig('green', greenConfig, setGreenConfig)}
              {renderColorConfig('blue', blueConfig, setBlueConfig)}
              {renderColorConfig('white', whiteConfig, setWhiteConfig)}
            </Box>
          </Paper>

          <Paper elevation={2} sx={{ p: 3, textAlign: 'center' }}>
            <Button
              variant="contained"
              size="large"
              onClick={handleRunSimulation}
              disabled={loading || !csvContent}
              startIcon={<PlayArrowIcon />}
              fullWidth
              sx={{ maxWidth: 400 }}
            >
              Run Simulation
            </Button>
          </Paper>

          {simulationResults && (
            <>
              <Paper elevation={2} sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom>
                  LED Power Visualization
                </Typography>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  Time: {currentResult?.timestamp.toFixed(3)}s (Sample {currentTimeIndex + 1} of {simulationResults.length})
                </Typography>

                <Box sx={{ mt: 3 }}>
                  {currentResult && (
                    <>
                      {renderColorBar('red', currentResult.red)}
                      {renderColorBar('green', currentResult.green)}
                      {renderColorBar('blue', currentResult.blue)}
                      {renderColorBar('white', currentResult.white)}
                    </>
                  )}
                </Box>
              </Paper>

              <Paper elevation={2} sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom>
                  Time Control
                </Typography>
                <Slider
                  value={currentTimeIndex}
                  onChange={handleTimeChange}
                  min={0}
                  max={maxTimeIndex}
                  step={1}
                  marks={[
                    { value: 0, label: '0s' },
                    { value: maxTimeIndex, label: `${simulationResults[maxTimeIndex]?.timestamp.toFixed(1)}s` },
                  ]}
                  valueLabelDisplay="auto"
                  valueLabelFormat={(value) => `${simulationResults[value]?.timestamp.toFixed(3)}s`}
                />
              </Paper>

              <Card elevation={2}>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Simulation Statistics
                  </Typography>
                  <Stack spacing={1}>
                    <Typography variant="body2">
                      <strong>Total Samples:</strong> {simulationResults.length}
                    </Typography>
                    <Typography variant="body2">
                      <strong>Duration:</strong> {simulationResults[simulationResults.length - 1]?.timestamp.toFixed(3)}s
                    </Typography>
                    <Typography variant="body2">
                      <strong>Sample Rate:</strong>{' '}
                      {(simulationResults.length / simulationResults[simulationResults.length - 1]?.timestamp).toFixed(1)} Hz
                    </Typography>
                  </Stack>
                </CardContent>
              </Card>
            </>
          )}
        </Box>
      </Container>

      <Backdrop sx={{ color: '#fff', zIndex: (theme) => theme.zIndex.drawer + 1 }} open={loading}>
        <CircularProgress color="inherit" />
      </Backdrop>

      <Snackbar open={snackbarOpen} autoHideDuration={6000} onClose={() => setSnackbarOpen(false)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        <Alert onClose={() => setSnackbarOpen(false)} severity="error" sx={{ width: '100%' }}>
          {error}
        </Alert>
      </Snackbar>
    </>
  );
}
