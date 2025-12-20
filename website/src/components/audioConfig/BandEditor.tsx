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
  const handleScalingChange = (value: string) => {
    const num = parseFloat(value);
    if (!isNaN(num) && num > 0) {
      onChange({ ...bandConfig, scaling_factor: num });
    }
  };

  const handleNoiseChange = (value: string) => {
    const num = parseFloat(value);
    if (!isNaN(num) && num >= 0) {
      onChange({ ...bandConfig, noise_threshold: num });
    }
  };

  return (
    <Card sx={{ borderLeft: `4px solid ${color}`, height: '100%' }}>
      <CardContent>
        <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mb: 2 }}>
          {label}
        </Typography>
        <Stack spacing={2}>
          <TextField
            label="Scaling Factor"
            type="text"
            value={bandConfig.scaling_factor.toExponential(6)}
            onChange={(e) => handleScalingChange(e.target.value)}
            size="small"
            fullWidth
            helperText="Converts raw magnitude to 0-1 range"
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
