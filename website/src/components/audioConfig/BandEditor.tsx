import { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Slider,
  Stack,
} from '@mui/material';
import type { AudioTuningBandConfig } from '../../types/api';

interface BandEditorProps {
  label: string;
  bandConfig: AudioTuningBandConfig;
  onChange: (config: AudioTuningBandConfig) => void;
  color: string;
}

export function BandEditor({ label, bandConfig, onChange, color }: BandEditorProps) {
  const [maxMagnitudeInput, setMaxMagnitudeInput] = useState<string | null>(null);

  const handleMaxMagnitudeChange = (value: string) => {
    setMaxMagnitudeInput(value);
    const num = parseFloat(value);
    if (!isNaN(num) && num > 0) {
      onChange({ ...bandConfig, max_magnitude: num });
    }
  };

  const handleMaxMagnitudeBlur = () => {
    setMaxMagnitudeInput(null);
  };

  const handleNoiseChange = (value: string) => {
    const num = parseFloat(value);
    if (!isNaN(num) && num >= 0) {
      onChange({ ...bandConfig, noise_threshold: num });
    }
  };

  const formatLargeNumber = (num: number): string => {
    if (num >= 1e6) return `${(num / 1e6).toFixed(2)}e6`;
    if (num >= 1e3) return `${(num / 1e3).toFixed(1)}e3`;
    return num.toFixed(0);
  };

  return (
    <Card sx={{ borderLeft: `4px solid ${color}`, height: '100%' }}>
      <CardContent>
        <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mb: 2 }}>
          {label}
        </Typography>
        <Stack spacing={2}>
          <TextField
            label="Max Magnitude"
            type="text"
            value={maxMagnitudeInput ?? formatLargeNumber(bandConfig.max_magnitude)}
            onChange={(e) => handleMaxMagnitudeChange(e.target.value)}
            onBlur={handleMaxMagnitudeBlur}
            size="small"
            fullWidth
            helperText="Raw value that maps to power 1.0"
          />
          <TextField
            label="Noise Threshold"
            type="number"
            value={bandConfig.noise_threshold}
            onChange={(e) => handleNoiseChange(e.target.value)}
            size="small"
            fullWidth
            helperText="Filter out background noise"
          />
          <Box>
            <Typography variant="body2" gutterBottom>
              Min/Max Power: {bandConfig.min_power_value.toFixed(2)} - {bandConfig.max_power_value.toFixed(2)}
            </Typography>
            <Slider
              value={[bandConfig.min_power_value, bandConfig.max_power_value]}
              onChange={(_, value) => {
                const [min, max] = value as number[];
                onChange({ ...bandConfig, min_power_value: min, max_power_value: max });
              }}
              min={0}
              max={1}
              step={0.05}
              valueLabelDisplay="auto"
            />
          </Box>
          <Box>
            <Typography variant="body2" gutterBottom>
              Smoothing: {bandConfig.smoothing.toFixed(2)}
            </Typography>
            <Slider
              value={bandConfig.smoothing}
              onChange={(_, value) => onChange({ ...bandConfig, smoothing: value as number })}
              min={0}
              max={1}
              step={0.05}
              valueLabelDisplay="auto"
            />
          </Box>
        </Stack>
      </CardContent>
    </Card>
  );
}
