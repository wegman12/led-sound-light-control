import { useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  FormGroup,
  FormControlLabel,
  Checkbox,
  ToggleButtonGroup,
  ToggleButton,
  Stack,
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
import type { ChartDataPoint } from '../../hooks/useAudioStream';

const TIME_WINDOW_SECONDS = 30;

interface VisibleChannels {
  bass: boolean;
  midLow: boolean;
  midHigh: boolean;
  treble: boolean;
}

interface AudioStreamChartProps {
  chartData: ChartDataPoint[];
  viewMode: 'raw' | 'processed' | 'both';
  visibleChannels: VisibleChannels;
  onViewModeChange: (mode: 'raw' | 'processed' | 'both') => void;
  onToggleChannel: (channel: keyof VisibleChannels) => void;
}

export function AudioStreamChart({
  chartData,
  viewMode,
  visibleChannels,
  onViewModeChange,
  onToggleChannel,
}: AudioStreamChartProps) {
  const { xMin, xMax } = useMemo(() => {
    if (chartData.length === 0) {
      return { xMin: 0, xMax: TIME_WINDOW_SECONDS };
    }
    const latestTime = chartData[chartData.length - 1].time;
    return {
      xMin: Math.max(0, latestTime - TIME_WINDOW_SECONDS),
      xMax: Math.max(TIME_WINDOW_SECONDS, latestTime),
    };
  }, [chartData]);

  const formatTime = (value: number) => `${value.toFixed(1)}s`;

  return (
    <Paper sx={{ p: 2 }}>
      <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2, flexWrap: 'wrap', gap: 1 }}>
        <Box>
          <Typography variant="body2" sx={{ fontWeight: 'medium', mb: 0.5 }}>
            View Mode:
          </Typography>
          <ToggleButtonGroup
            value={viewMode}
            exclusive
            onChange={(_, newMode) => newMode && onViewModeChange(newMode)}
            size="small"
          >
            <ToggleButton value="raw">Raw</ToggleButton>
            <ToggleButton value="processed">Processed</ToggleButton>
            <ToggleButton value="both">Both</ToggleButton>
          </ToggleButtonGroup>
        </Box>

        <Box>
          <Typography variant="body2" sx={{ fontWeight: 'medium', mb: 0.5 }}>
            Channels:
          </Typography>
          <FormGroup row>
            <FormControlLabel
              control={
                <Checkbox
                  checked={visibleChannels.bass}
                  onChange={() => onToggleChannel('bass')}
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
                  onChange={() => onToggleChannel('midLow')}
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
                  onChange={() => onToggleChannel('midHigh')}
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
                  onChange={() => onToggleChannel('treble')}
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
          Points: {chartData.length} | Window: {TIME_WINDOW_SECONDS}s
        </Typography>
      </Stack>

      {/* Raw Audio Chart */}
      {viewMode !== 'processed' && (
        <Box sx={{ mb: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            Raw Audio Magnitude
          </Typography>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#444" />
              <XAxis
                dataKey="time"
                type="number"
                domain={[xMin, xMax]}
                tickFormatter={formatTime}
                allowDataOverflow={true}
              />
              <YAxis tickFormatter={(value) => value.toExponential(1)} />
              <Tooltip
                labelFormatter={formatTime}
                formatter={(value: number) => value.toFixed(0)}
              />
              <Legend />
              {visibleChannels.bass && (
                <Line type="monotone" dataKey="bassRaw" stroke="#ff4444" strokeWidth={2} dot={false} name="Bass" isAnimationActive={false} />
              )}
              {visibleChannels.midLow && (
                <Line type="monotone" dataKey="midLowRaw" stroke="#44ff44" strokeWidth={2} dot={false} name="Mid-Low" isAnimationActive={false} />
              )}
              {visibleChannels.midHigh && (
                <Line type="monotone" dataKey="midHighRaw" stroke="#4444ff" strokeWidth={2} dot={false} name="Mid-High" isAnimationActive={false} />
              )}
              {visibleChannels.treble && (
                <Line type="monotone" dataKey="trebleRaw" stroke="#ff44ff" strokeWidth={2} dot={false} name="Treble" isAnimationActive={false} />
              )}
            </LineChart>
          </ResponsiveContainer>
        </Box>
      )}

      {/* Processed Audio Chart */}
      {viewMode !== 'raw' && (
        <Box>
          <Typography variant="subtitle2" gutterBottom>
            Processed LED Power (0-1)
          </Typography>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#444" />
              <XAxis
                dataKey="time"
                type="number"
                domain={[xMin, xMax]}
                tickFormatter={formatTime}
                allowDataOverflow={true}
              />
              <YAxis domain={[0, 1]} />
              <Tooltip
                labelFormatter={formatTime}
                formatter={(value: number) => value.toFixed(3)}
              />
              <Legend />
              {visibleChannels.bass && (
                <Line type="monotone" dataKey="bassProcessed" stroke="#ff4444" strokeWidth={2} dot={false} name="Bass" isAnimationActive={false} />
              )}
              {visibleChannels.midLow && (
                <Line type="monotone" dataKey="midLowProcessed" stroke="#44ff44" strokeWidth={2} dot={false} name="Mid-Low" isAnimationActive={false} />
              )}
              {visibleChannels.midHigh && (
                <Line type="monotone" dataKey="midHighProcessed" stroke="#4444ff" strokeWidth={2} dot={false} name="Mid-High" isAnimationActive={false} />
              )}
              {visibleChannels.treble && (
                <Line type="monotone" dataKey="trebleProcessed" stroke="#ff44ff" strokeWidth={2} dot={false} name="Treble" isAnimationActive={false} />
              )}
            </LineChart>
          </ResponsiveContainer>
        </Box>
      )}
    </Paper>
  );
}
