import { useState, useEffect, useRef, useCallback } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Typography,
  TextField,
  Alert,
  Snackbar,
  Paper,
  Divider,
  Stack,
  ToggleButtonGroup,
  ToggleButton,
  Chip,
  FormGroup,
  FormControlLabel,
  Checkbox,
} from '@mui/material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import {
  getTuningConfig,
  updateTuningConfig,
  saveTuningConfig,
  startAudioStream,
  stopAudioStream,
  getWebSocketUrl,
} from '../services';
import type { AudioTuningConfig, AudioStreamData } from '../types/api';

interface ChartDataPoint {
  time: number; // Time in seconds
  bassRaw: number;
  midLowRaw: number;
  midHighRaw: number;
  trebleRaw: number;
  bassProcessed: number;
  midLowProcessed: number;
  midHighProcessed: number;
  trebleProcessed: number;
}

const TIME_WINDOW_SECONDS = 30; // Show 30 seconds of data
const MAX_DATA_POINTS = 1200; // 30 seconds at 40 Hz = 1200 points

export function AudioTuningPage() {
  const [config, setConfig] = useState<AudioTuningConfig | null>(null);
  const [chartData, setChartData] = useState<ChartDataPoint[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'raw' | 'processed' | 'both'>('processed');
  const [visibleChannels, setVisibleChannels] = useState({
    bass: true,
    midLow: true,
    midHigh: true,
    treble: true,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const startTimeRef = useRef<number>(Date.now());
  const isStreamingRef = useRef<boolean>(false);
  const configLoadedRef = useRef<boolean>(false);

  // Keep ref in sync with state
  useEffect(() => {
    isStreamingRef.current = isStreaming;
  }, [isStreaming]);

  // Load initial configuration
  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      const loadedConfig = await getTuningConfig();
      setConfig(loadedConfig);
      configLoadedRef.current = true;
    } catch (err) {
      setError('Failed to load configuration: ' + (err as Error).message);
    }
  };

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    const wsUrl = getWebSocketUrl('/api/audio/tuning/stream');
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
      startTimeRef.current = Date.now();
    };

    ws.onmessage = (event) => {
      try {
        const data: AudioStreamData = JSON.parse(event.data);

        // Only update config from WebSocket on initial load (if we haven't loaded it yet)
        if (data.config && !configLoadedRef.current) {
          setConfig(data.config);
          configLoadedRef.current = true;
        }

        // Only add data point to chart if streaming is active (use ref to avoid stale closure)
        if (data.raw && data.processed && isStreamingRef.current) {
          const timeSeconds = (Date.now() - startTimeRef.current) / 1000;

          setChartData((prev) => {
            const newPoint: ChartDataPoint = {
              time: timeSeconds,
              bassRaw: data.raw.bass,
              midLowRaw: data.raw.mid_low,
              midHighRaw: data.raw.mid_high,
              trebleRaw: data.raw.treble,
              bassProcessed: data.processed.bass,
              midLowProcessed: data.processed.mid_low,
              midHighProcessed: data.processed.mid_high,
              trebleProcessed: data.processed.treble,
            };

            const newData = [...prev, newPoint];
            // Keep only last MAX_DATA_POINTS
            return newData.slice(-MAX_DATA_POINTS);
          });
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    ws.onerror = (event) => {
      console.error('WebSocket error:', event);
      setError('WebSocket connection error');
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      setIsConnected(false);
    };

    wsRef.current = ws;
  }, []);

  const disconnectWebSocket = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
      setIsConnected(false);
    }
  }, []);

  useEffect(() => {
    // Connect WebSocket on mount
    connectWebSocket();

    // Cleanup on unmount
    return () => {
      disconnectWebSocket();
    };
  }, [connectWebSocket, disconnectWebSocket]);

  const handleStartStreaming = async () => {
    try {
      await startAudioStream();
      setIsStreaming(true);
      setChartData([]); // Clear previous data
      startTimeRef.current = Date.now();
      setSuccess('Audio streaming started');
    } catch (err) {
      setError('Failed to start streaming: ' + (err as Error).message);
    }
  };

  const handleStopStreaming = async () => {
    try {
      await stopAudioStream();
      setIsStreaming(false);
      setSuccess('Audio streaming stopped');
    } catch (err) {
      setError('Failed to stop streaming: ' + (err as Error).message);
    }
  };

  const handleUpdateConfig = async () => {
    if (!config) return;

    try {
      await updateTuningConfig(config);
      setSuccess('Configuration updated');
    } catch (err) {
      setError('Failed to update configuration: ' + (err as Error).message);
    }
  };

  const handleSaveConfig = async () => {
    try {
      await saveTuningConfig();
      setSuccess('Configuration saved to file');
    } catch (err) {
      setError('Failed to save configuration: ' + (err as Error).message);
    }
  };

  const handleConfigChange = (
    section: 'bass' | 'mid_low' | 'mid_high' | 'treble',
    field: keyof AudioTuningConfig['bass'],
    value: number,
  ) => {
    if (!config) return;

    setConfig({
      ...config,
      [section]: {
        ...config[section],
        [field]: value,
      },
    });
  };

  const handleCutoffChange = (
    field: 'bass_cutoff' | 'mid_high_cutoff' | 'treble_cutoff',
    value: number,
  ) => {
    if (!config) return;

    setConfig({
      ...config,
      [field]: value,
    });
  };

  // Calculate X-axis domain for fixed time window
  const getXAxisDomain = (): [number, number] => {
    if (chartData.length === 0) {
      return [0, TIME_WINDOW_SECONDS];
    }

    const latestTime = chartData[chartData.length - 1]?.time || 0;

    if (latestTime < TIME_WINDOW_SECONDS) {
      // Still filling the initial window
      return [0, TIME_WINDOW_SECONDS];
    }

    // Rolling window
    return [latestTime - TIME_WINDOW_SECONDS, latestTime];
  };

  // Format time for display
  const formatTime = (time: number) => {
    return time.toFixed(1) + 's';
  };

  // Toggle channel visibility
  const handleToggleChannel = (channel: keyof typeof visibleChannels) => {
    setVisibleChannels((prev) => ({
      ...prev,
      [channel]: !prev[channel],
    }));
  };

  if (!config) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography>Loading configuration...</Typography>
      </Box>
    );
  }

  const [xMin, xMax] = getXAxisDomain();

  return (
    <Box sx={{ p: 3, maxWidth: '1600px', mx: 'auto' }}>
      <Typography variant="h4" gutterBottom>
        Audio Tuning
      </Typography>

      <Typography variant="body2" color="text.secondary" gutterBottom sx={{ mb: 3 }}>
        Tune audio parameters in real-time and visualize the results. Changes apply immediately to the live audio stream.
      </Typography>

      {/* Control Panel */}
      <Paper sx={{ p: 2, mb: 3, bgcolor: 'background.default' }}>
        <Stack direction="row" spacing={2} sx={{ mb: 2, flexWrap: 'wrap', gap: 1 }}>
          <Button
            variant="contained"
            color={isStreaming ? 'error' : 'primary'}
            onClick={isStreaming ? handleStopStreaming : handleStartStreaming}
            size="large"
          >
            {isStreaming ? 'Stop Streaming' : 'Start Streaming'}
          </Button>
          <Button
            variant="contained"
            onClick={handleUpdateConfig}
          >
            Apply Configuration
          </Button>
          <Button
            variant="contained"
            color="success"
            onClick={handleSaveConfig}
          >
            Save to File
          </Button>
          <Box sx={{ flexGrow: 1 }} />
          <Chip
            label={isConnected ? 'Connected' : 'Disconnected'}
            color={isConnected ? 'success' : 'error'}
            variant="outlined"
          />
        </Stack>

        <Divider sx={{ my: 2 }} />

        <Stack direction="row" spacing={2} alignItems="center" sx={{ flexWrap: 'wrap', gap: 2 }}>
          <Box>
            <Typography variant="body2" sx={{ fontWeight: 'medium', mb: 0.5 }}>
              View Mode:
            </Typography>
            <ToggleButtonGroup
              value={viewMode}
              exclusive
              onChange={(_, newMode) => newMode && setViewMode(newMode)}
              size="small"
            >
              <ToggleButton value="raw">Raw</ToggleButton>
              <ToggleButton value="processed">Processed</ToggleButton>
              <ToggleButton value="both">Both</ToggleButton>
            </ToggleButtonGroup>
          </Box>

          <Divider orientation="vertical" flexItem />

          <Box>
            <Typography variant="body2" sx={{ fontWeight: 'medium', mb: 0.5 }}>
              Channels:
            </Typography>
            <FormGroup row>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={visibleChannels.bass}
                    onChange={() => handleToggleChannel('bass')}
                    size="small"
                    sx={{ color: '#ff4444', '&.Mui-checked': { color: '#ff4444' } }}
                  />
                }
                label={<Typography variant="body2">Bass</Typography>}
              />
              <FormControlLabel
                control={
                  <Checkbox
                    checked={visibleChannels.midLow}
                    onChange={() => handleToggleChannel('midLow')}
                    size="small"
                    sx={{ color: '#44ff44', '&.Mui-checked': { color: '#44ff44' } }}
                  />
                }
                label={<Typography variant="body2">Mid-Low</Typography>}
              />
              <FormControlLabel
                control={
                  <Checkbox
                    checked={visibleChannels.midHigh}
                    onChange={() => handleToggleChannel('midHigh')}
                    size="small"
                    sx={{ color: '#4444ff', '&.Mui-checked': { color: '#4444ff' } }}
                  />
                }
                label={<Typography variant="body2">Mid-High</Typography>}
              />
              <FormControlLabel
                control={
                  <Checkbox
                    checked={visibleChannels.treble}
                    onChange={() => handleToggleChannel('treble')}
                    size="small"
                    sx={{ color: '#ff44ff', '&.Mui-checked': { color: '#ff44ff' } }}
                  />
                }
                label={<Typography variant="body2">Treble</Typography>}
              />
            </FormGroup>
          </Box>

          <Box sx={{ flexGrow: 1 }} />

          <Typography variant="body2" color="text.secondary">
            Data points: {chartData.length} | Window: {TIME_WINDOW_SECONDS}s
          </Typography>
        </Stack>
      </Paper>

      {/* Charts */}
      {viewMode !== 'processed' && (
        <Paper sx={{ p: 2, mb: 3 }}>
          <Typography variant="h6" gutterBottom>
            Raw Audio Values
          </Typography>
          <ResponsiveContainer width="100%" height={350}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#444" />
              <XAxis
                dataKey="time"
                type="number"
                domain={[xMin, xMax]}
                tickFormatter={formatTime}
                label={{ value: 'Time (seconds)', position: 'insideBottom', offset: -5 }}
                allowDataOverflow={true}
              />
              <YAxis
                label={{ value: 'Magnitude', angle: -90, position: 'insideLeft' }}
                tickFormatter={(value) => value.toExponential(1)}
              />
              <Tooltip
                labelFormatter={formatTime}
                formatter={(value: number) => value.toFixed(0)}
              />
              <Legend />
              {visibleChannels.bass && (
                <Line
                  type="monotone"
                  dataKey="bassRaw"
                  stroke="#ff4444"
                  strokeWidth={2}
                  dot={false}
                  name="Bass"
                  isAnimationActive={false}
                  connectNulls={false}
                />
              )}
              {visibleChannels.midLow && (
                <Line
                  type="monotone"
                  dataKey="midLowRaw"
                  stroke="#44ff44"
                  strokeWidth={2}
                  dot={false}
                  name="Mid-Low"
                  isAnimationActive={false}
                  connectNulls={false}
                />
              )}
              {visibleChannels.midHigh && (
                <Line
                  type="monotone"
                  dataKey="midHighRaw"
                  stroke="#4444ff"
                  strokeWidth={2}
                  dot={false}
                  name="Mid-High"
                  isAnimationActive={false}
                  connectNulls={false}
                />
              )}
              {visibleChannels.treble && (
                <Line
                  type="monotone"
                  dataKey="trebleRaw"
                  stroke="#ff44ff"
                  strokeWidth={2}
                  dot={false}
                  name="Treble"
                  isAnimationActive={false}
                  connectNulls={false}
                />
              )}
            </LineChart>
          </ResponsiveContainer>
        </Paper>
      )}

      {viewMode !== 'raw' && (
        <Paper sx={{ p: 2, mb: 3 }}>
          <Typography variant="h6" gutterBottom>
            Processed LED Power (0-1)
          </Typography>
          <ResponsiveContainer width="100%" height={350}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#444" />
              <XAxis
                dataKey="time"
                type="number"
                domain={[xMin, xMax]}
                tickFormatter={formatTime}
                label={{ value: 'Time (seconds)', position: 'insideBottom', offset: -5 }}
                allowDataOverflow={true}
              />
              <YAxis
                domain={[0, 1]}
                label={{ value: 'LED Power', angle: -90, position: 'insideLeft' }}
              />
              <Tooltip
                labelFormatter={formatTime}
                formatter={(value: number) => value.toFixed(3)}
              />
              <Legend />
              {visibleChannels.bass && (
                <Line
                  type="monotone"
                  dataKey="bassProcessed"
                  stroke="#ff4444"
                  strokeWidth={2}
                  dot={false}
                  name="Bass"
                  isAnimationActive={false}
                  connectNulls={false}
                />
              )}
              {visibleChannels.midLow && (
                <Line
                  type="monotone"
                  dataKey="midLowProcessed"
                  stroke="#44ff44"
                  strokeWidth={2}
                  dot={false}
                  name="Mid-Low"
                  isAnimationActive={false}
                  connectNulls={false}
                />
              )}
              {visibleChannels.midHigh && (
                <Line
                  type="monotone"
                  dataKey="midHighProcessed"
                  stroke="#4444ff"
                  strokeWidth={2}
                  dot={false}
                  name="Mid-High"
                  isAnimationActive={false}
                  connectNulls={false}
                />
              )}
              {visibleChannels.treble && (
                <Line
                  type="monotone"
                  dataKey="trebleProcessed"
                  stroke="#ff44ff"
                  strokeWidth={2}
                  dot={false}
                  name="Treble"
                  isAnimationActive={false}
                  connectNulls={false}
                />
              )}
            </LineChart>
          </ResponsiveContainer>
        </Paper>
      )}

      {/* Frequency Band Cutoffs */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Frequency Band Cutoffs (Hz)
          </Typography>
          <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', gap: 2 }}>
            <TextField
              label="Bass Cutoff"
              type="number"
              value={config.bass_cutoff}
              onChange={(e) =>
                handleCutoffChange('bass_cutoff', parseFloat(e.target.value))
              }
              helperText="0 - Bass Cutoff Hz"
              sx={{ flex: '1 1 250px', minWidth: '200px' }}
            />
            <TextField
              label="Mid-High Cutoff"
              type="number"
              value={config.mid_high_cutoff}
              onChange={(e) =>
                handleCutoffChange('mid_high_cutoff', parseFloat(e.target.value))
              }
              helperText="Bass - Mid-High Hz (mid-low)"
              sx={{ flex: '1 1 250px', minWidth: '200px' }}
            />
            <TextField
              label="Treble Cutoff"
              type="number"
              value={config.treble_cutoff}
              onChange={(e) =>
                handleCutoffChange('treble_cutoff', parseFloat(e.target.value))
              }
              helperText="Mid-High - Treble Hz (treble above)"
              sx={{ flex: '1 1 250px', minWidth: '200px' }}
            />
          </Stack>
        </CardContent>
      </Card>

      {/* Band Configuration Cards */}
      <Typography variant="h5" gutterBottom sx={{ mt: 4, mb: 2 }}>
        Band Configuration
      </Typography>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: 3 }}>
        {(['bass', 'mid_low', 'mid_high', 'treble'] as const).map((band) => (
          <Card key={band} sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom sx={{ textTransform: 'uppercase' }}>
                {band.replace('_', '-')}
              </Typography>
              <Divider sx={{ mb: 2 }} />
              <Stack spacing={2.5}>
                <TextField
                  fullWidth
                  label="Scaling Factor"
                  type="number"
                  inputProps={{ step: 0.000001 }}
                  value={config[band].scaling_factor}
                  onChange={(e) =>
                    handleConfigChange(
                      band,
                      'scaling_factor',
                      parseFloat(e.target.value),
                    )
                  }
                  helperText="Converts raw magnitude to 0-1"
                />
                <TextField
                  fullWidth
                  label="Noise Threshold"
                  type="number"
                  value={config[band].noise_threshold}
                  onChange={(e) =>
                    handleConfigChange(
                      band,
                      'noise_threshold',
                      parseFloat(e.target.value),
                    )
                  }
                  helperText="Filters background noise"
                />
                <Box sx={{ display: 'flex', gap: 2 }}>
                  <TextField
                    fullWidth
                    label="Min Power"
                    type="number"
                    inputProps={{ min: 0, max: 1, step: 0.1 }}
                    value={config[band].min_power_value}
                    onChange={(e) =>
                      handleConfigChange(
                        band,
                        'min_power_value',
                        parseFloat(e.target.value),
                      )
                    }
                    helperText="LED min (0-1)"
                  />
                  <TextField
                    fullWidth
                    label="Max Power"
                    type="number"
                    inputProps={{ min: 0, max: 1, step: 0.1 }}
                    value={config[band].max_power_value}
                    onChange={(e) =>
                      handleConfigChange(
                        band,
                        'max_power_value',
                        parseFloat(e.target.value),
                      )
                    }
                    helperText="LED max (0-1)"
                  />
                </Box>
                <TextField
                  fullWidth
                  label="Smoothing"
                  type="number"
                  inputProps={{ min: 0, max: 1, step: 0.1 }}
                  value={config[band].smoothing}
                  onChange={(e) =>
                    handleConfigChange(
                      band,
                      'smoothing',
                      parseFloat(e.target.value),
                    )
                  }
                  helperText="Exponential smoothing (0-1, higher = more smoothing)"
                />
              </Stack>
            </CardContent>
          </Card>
        ))}
      </Box>

      {/* Error/Success Snackbars */}
      <Snackbar
        open={!!error}
        autoHideDuration={6000}
        onClose={() => setError(null)}
      >
        <Alert severity="error" onClose={() => setError(null)}>
          {error}
        </Alert>
      </Snackbar>

      <Snackbar
        open={!!success}
        autoHideDuration={3000}
        onClose={() => setSuccess(null)}
      >
        <Alert severity="success" onClose={() => setSuccess(null)}>
          {success}
        </Alert>
      </Snackbar>
    </Box>
  );
}
