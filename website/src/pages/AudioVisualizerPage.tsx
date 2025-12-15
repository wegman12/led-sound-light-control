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
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
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
  scalingFactor: number;
  noiseThreshold: number;
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
    scalingFactor: 0.000001,
    noiseThreshold: 92565436,
  });

  const [greenConfig, setGreenConfig] = useState<ColorBandConfig>({
    enabled: true,
    frequencyBand: 'mid-low',
    minPower: 0.1,
    maxPower: 1.0,
    smoothing: 0.3,
    scalingFactor: 0.000001,
    noiseThreshold: 18541891,
  });

  const [blueConfig, setBlueConfig] = useState<ColorBandConfig>({
    enabled: true,
    frequencyBand: 'mid-high',
    minPower: 0.1,
    maxPower: 1.0,
    smoothing: 0.3,
    scalingFactor: 0.000002,
    noiseThreshold: 10913229,
  });

  const [whiteConfig, setWhiteConfig] = useState<ColorBandConfig>({
    enabled: true,
    frequencyBand: 'treble',
    minPower: 0.0,
    maxPower: 0.8,
    smoothing: 0.1,
    scalingFactor: 0.000027,
    noiseThreshold: 2258769,
  });

  // Chart display controls
  const [visibleColors, setVisibleColors] = useState({
    red: true,
    green: true,
    blue: true,
    white: true,
  });

  const [timeRange, setTimeRange] = useState<[number, number]>([0, 0]);

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
    config: ColorBandConfig
  ): AudioModulatorConfig => {
    return {
      frequency_band: config.frequencyBand,
      min_power_value: config.minPower,
      max_power_value: config.maxPower,
      scaling_factor: config.scalingFactor,
      noise_threshold: config.noiseThreshold,
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
            config: createAudioBehaviorConfig(config),
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

      // Initialize time range to full duration
      if (response.results.length > 0) {
        setTimeRange([0, response.results[response.results.length - 1].timestamp]);
      }
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

  const handleTimeRangeChange = (_event: Event, value: number | number[]) => {
    const [start, end] = value as number[];
    setTimeRange([start, end]);
  };

  const toggleColorVisibility = (color: keyof typeof visibleColors) => {
    setVisibleColors((prev) => ({ ...prev, [color]: !prev[color] }));
  };

  // Filter simulation results by time range
  const filteredResults = simulationResults?.filter(
    (result) => result.timestamp >= timeRange[0] && result.timestamp <= timeRange[1]
  ) || [];

  const currentResult = simulationResults?.[currentTimeIndex];
  const maxTimeIndex = (simulationResults?.length || 1) - 1;
  const maxTimestamp = simulationResults?.[maxTimeIndex]?.timestamp || 0;

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
    const bandKey = config.frequencyBand.replace('-', '_') as keyof AudioConfigResponse;
    const bandConfig = audioConfig?.[bandKey];

    return (
      <Paper
        key={color}
        elevation={1}
        sx={{
          p: 2,
          opacity: config.enabled ? 1 : 0.6,
          border: config.enabled ? `2px solid ${colorInfo.color}` : '2px solid transparent',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Typography variant="h6" sx={{ color: colorInfo.color, fontWeight: 'bold' }}>
            {colorInfo.label}
          </Typography>
          <FormControlLabel
            control={<Switch checked={config.enabled} onChange={(e) => setConfig({ ...config, enabled: e.target.checked })} />}
            label="Enabled"
          />
        </Box>

        {config.enabled && (
          <Stack spacing={2}>
            {/* Frequency Band */}
            <FormControl fullWidth size="small">
              <InputLabel>Frequency Band</InputLabel>
              <Select
                value={config.frequencyBand}
                label="Frequency Band"
                onChange={(e) => setConfig({ ...config, frequencyBand: e.target.value as FrequencyBand })}
              >
                {FREQUENCY_BANDS.map((band) => (
                  <MenuItem key={band.value} value={band.value}>
                    {band.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            {/* Min Power */}
            <Box>
              <Typography variant="body2" gutterBottom>
                Min Power: {(config.minPower * 100).toFixed(0)}%
              </Typography>
              <Slider
                value={config.minPower}
                onChange={(_e, value) => setConfig({ ...config, minPower: value as number })}
                min={0}
                max={1}
                step={0.05}
                valueLabelDisplay="auto"
                valueLabelFormat={(value) => `${(value * 100).toFixed(0)}%`}
                marks={[
                  { value: 0, label: '0%' },
                  { value: 0.5, label: '50%' },
                  { value: 1, label: '100%' },
                ]}
              />
            </Box>

            {/* Max Power */}
            <Box>
              <Typography variant="body2" gutterBottom>
                Max Power: {(config.maxPower * 100).toFixed(0)}%
              </Typography>
              <Slider
                value={config.maxPower}
                onChange={(_e, value) => setConfig({ ...config, maxPower: value as number })}
                min={0}
                max={1}
                step={0.05}
                valueLabelDisplay="auto"
                valueLabelFormat={(value) => `${(value * 100).toFixed(0)}%`}
                marks={[
                  { value: 0, label: '0%' },
                  { value: 0.5, label: '50%' },
                  { value: 1, label: '100%' },
                ]}
              />
            </Box>

            {/* Smoothing */}
            <Box>
              <Typography variant="body2" gutterBottom>
                Smoothing: {(config.smoothing * 100).toFixed(0)}% (Higher = Smoother, Slower Response)
              </Typography>
              <Slider
                value={config.smoothing}
                onChange={(_e, value) => setConfig({ ...config, smoothing: value as number })}
                min={0}
                max={1}
                step={0.05}
                valueLabelDisplay="auto"
                valueLabelFormat={(value) => `${(value * 100).toFixed(0)}%`}
                marks={[
                  { value: 0, label: 'None' },
                  { value: 0.3, label: 'Low' },
                  { value: 0.6, label: 'Med' },
                  { value: 1, label: 'High' },
                ]}
              />
            </Box>

            {/* Advanced Settings */}
            <Box sx={{ mt: 2, p: 2, bgcolor: 'rgba(0,0,0,0.05)', borderRadius: 1 }}>
              <Typography variant="body2" fontWeight="bold" gutterBottom>
                Advanced Settings
              </Typography>

              {/* Scaling Factor */}
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" display="block" gutterBottom>
                  Scaling Factor: {config.scalingFactor.toExponential(2)}
                  {bandConfig && ` (Default: ${bandConfig.scaling_factor.toExponential(2)})`}
                </Typography>
                <Slider
                  value={Math.log10(config.scalingFactor)}
                  onChange={(_e, value) => setConfig({ ...config, scalingFactor: Math.pow(10, value as number) })}
                  min={-8}
                  max={-3}
                  step={0.1}
                  valueLabelDisplay="auto"
                  valueLabelFormat={(value) => `${Math.pow(10, value).toExponential(1)}`}
                  marks={[
                    { value: -8, label: '1e-8' },
                    { value: -6, label: '1e-6' },
                    { value: -3, label: '1e-3' },
                  ]}
                />
                <Typography variant="caption" color="text.secondary">
                  Converts raw audio magnitude to 0-1 range. Higher = brighter for same input.
                </Typography>
              </Box>

              {/* Noise Threshold */}
              <Box>
                <Typography variant="caption" display="block" gutterBottom>
                  Noise Threshold: {config.noiseThreshold.toLocaleString()}
                  {bandConfig && ` (Default: ${bandConfig.noise_threshold.toLocaleString()})`}
                </Typography>
                <Slider
                  value={config.noiseThreshold}
                  onChange={(_e, value) => setConfig({ ...config, noiseThreshold: value as number })}
                  min={0}
                  max={bandConfig ? bandConfig.noise_threshold * 5 : 200000000}
                  step={bandConfig ? bandConfig.noise_threshold * 0.1 : 1000000}
                  valueLabelDisplay="auto"
                  valueLabelFormat={(value) => value.toLocaleString()}
                  marks={
                    bandConfig
                      ? [
                          { value: 0, label: '0' },
                          { value: bandConfig.noise_threshold, label: 'Default' },
                          { value: bandConfig.noise_threshold * 3, label: '3x' },
                        ]
                      : []
                  }
                />
                <Typography variant="caption" color="text.secondary">
                  Filters out background noise. Values below threshold are treated as zero. Higher = less sensitive.
                </Typography>
              </Box>
            </Box>
          </Stack>
        )}
      </Paper>
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

            {/* Quick Smoothing Presets */}
            <Box sx={{ mb: 3, p: 2, bgcolor: 'rgba(25, 118, 210, 0.08)', borderRadius: 1 }}>
              <Typography variant="subtitle2" gutterBottom fontWeight="bold">
                Quick Smoothing Presets (Apply to All Colors)
              </Typography>
              <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => {
                    const smoothing = 0.0;
                    setRedConfig((c) => ({ ...c, smoothing }));
                    setGreenConfig((c) => ({ ...c, smoothing }));
                    setBlueConfig((c) => ({ ...c, smoothing }));
                    setWhiteConfig((c) => ({ ...c, smoothing }));
                  }}
                >
                  No Smoothing (0%)
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => {
                    const smoothing = 0.3;
                    setRedConfig((c) => ({ ...c, smoothing }));
                    setGreenConfig((c) => ({ ...c, smoothing }));
                    setBlueConfig((c) => ({ ...c, smoothing }));
                    setWhiteConfig((c) => ({ ...c, smoothing }));
                  }}
                >
                  Low (30%)
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => {
                    const smoothing = 0.5;
                    setRedConfig((c) => ({ ...c, smoothing }));
                    setGreenConfig((c) => ({ ...c, smoothing }));
                    setBlueConfig((c) => ({ ...c, smoothing }));
                    setWhiteConfig((c) => ({ ...c, smoothing }));
                  }}
                >
                  Medium (50%)
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => {
                    const smoothing = 0.7;
                    setRedConfig((c) => ({ ...c, smoothing }));
                    setGreenConfig((c) => ({ ...c, smoothing }));
                    setBlueConfig((c) => ({ ...c, smoothing }));
                    setWhiteConfig((c) => ({ ...c, smoothing }));
                  }}
                >
                  High (70%)
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => {
                    const smoothing = 0.9;
                    setRedConfig((c) => ({ ...c, smoothing }));
                    setGreenConfig((c) => ({ ...c, smoothing }));
                    setBlueConfig((c) => ({ ...c, smoothing }));
                    setWhiteConfig((c) => ({ ...c, smoothing }));
                  }}
                >
                  Very High (90%)
                </Button>
              </Stack>
            </Box>

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

              <Paper elevation={2} sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom>
                  LED Power Over Time
                </Typography>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  Visualize the LED power values across the full sample duration
                </Typography>

                {/* Color toggles */}
                <Box sx={{ mt: 2, mb: 2, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={visibleColors.red}
                        onChange={() => toggleColorVisibility('red')}
                        size="small"
                      />
                    }
                    label={<Typography sx={{ color: COLOR_INFO.red.color, fontWeight: 'bold' }}>Red</Typography>}
                  />
                  <FormControlLabel
                    control={
                      <Switch
                        checked={visibleColors.green}
                        onChange={() => toggleColorVisibility('green')}
                        size="small"
                      />
                    }
                    label={<Typography sx={{ color: COLOR_INFO.green.color, fontWeight: 'bold' }}>Green</Typography>}
                  />
                  <FormControlLabel
                    control={
                      <Switch
                        checked={visibleColors.blue}
                        onChange={() => toggleColorVisibility('blue')}
                        size="small"
                      />
                    }
                    label={<Typography sx={{ color: COLOR_INFO.blue.color, fontWeight: 'bold' }}>Blue</Typography>}
                  />
                  <FormControlLabel
                    control={
                      <Switch
                        checked={visibleColors.white}
                        onChange={() => toggleColorVisibility('white')}
                        size="small"
                      />
                    }
                    label={<Typography sx={{ fontWeight: 'bold' }}>White</Typography>}
                  />
                </Box>

                {/* Time range selector */}
                <Box sx={{ mt: 3, mb: 3 }}>
                  <Typography variant="body2" gutterBottom>
                    Time Range: {timeRange[0].toFixed(1)}s - {timeRange[1].toFixed(1)}s
                  </Typography>
                  <Slider
                    value={timeRange}
                    onChange={handleTimeRangeChange}
                    min={0}
                    max={maxTimestamp}
                    step={0.1}
                    valueLabelDisplay="auto"
                    valueLabelFormat={(value) => `${value.toFixed(1)}s`}
                    marks={[
                      { value: 0, label: '0s' },
                      { value: maxTimestamp, label: `${maxTimestamp.toFixed(1)}s` },
                    ]}
                  />
                </Box>

                {/* Chart */}
                <ResponsiveContainer width="100%" height={400}>
                  <LineChart data={filteredResults}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="timestamp"
                      label={{ value: 'Time (s)', position: 'insideBottom', offset: -5 }}
                      tickFormatter={(value) => value.toFixed(1)}
                      domain={[timeRange[0], timeRange[1]]}
                    />
                    <YAxis
                      label={{ value: 'Power', angle: -90, position: 'insideLeft' }}
                      domain={[0, 1]}
                      tickFormatter={(value) => `${(value * 100).toFixed(0)}%`}
                    />
                    <Tooltip
                      formatter={(value: number) => `${(value * 100).toFixed(1)}%`}
                      labelFormatter={(label) => `Time: ${label.toFixed(3)}s`}
                    />
                    <Legend />
                    {visibleColors.red && (
                      <Line type="monotone" dataKey="red" stroke="#f44336" strokeWidth={2} dot={false} name="Red" />
                    )}
                    {visibleColors.green && (
                      <Line type="monotone" dataKey="green" stroke="#4caf50" strokeWidth={2} dot={false} name="Green" />
                    )}
                    {visibleColors.blue && (
                      <Line type="monotone" dataKey="blue" stroke="#2196f3" strokeWidth={2} dot={false} name="Blue" />
                    )}
                    {visibleColors.white && (
                      <Line type="monotone" dataKey="white" stroke="#9e9e9e" strokeWidth={2} dot={false} name="White" />
                    )}
                  </LineChart>
                </ResponsiveContainer>
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
