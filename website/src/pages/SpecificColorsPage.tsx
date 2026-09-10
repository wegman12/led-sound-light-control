import { useState } from 'react';
import {
  Container,
  Typography,
  Box,
  Backdrop,
  CircularProgress,
  Snackbar,
  Alert,
  Slider,
} from '@mui/material';
import PresetColorCard from '../components/PresetColorCard';
import { registerBehavior, turnLightsOn, ApiError } from '../services';
import type { ManagerConfig } from '../types/api';

interface PresetColor {
  name: string;
  red: number;
  green: number;
  blue: number;
  white: number;
}

const PRESET_COLORS: PresetColor[] = [
  { name: 'Red', red: 1, green: 0, blue: 0, white: 0 },
  { name: 'Green', red: 0, green: 1, blue: 0, white: 0 },
  { name: 'Blue', red: 0, green: 0, blue: 1, white: 0 },
  { name: 'Yellow', red: 1, green: 1, blue: 0, white: 0 },
  { name: 'Cyan', red: 0, green: 1, blue: 1, white: 0 },
  { name: 'Magenta', red: 1, green: 0, blue: 1, white: 0 },
  { name: 'White', red: 0, green: 0, blue: 0, white: 1 },
  { name: 'Warm White', red: 1, green: 0.7, blue: 0.3, white: 0.5 },
  { name: 'Orange', red: 1, green: 0.5, blue: 0, white: 0 },
  { name: 'Purple', red: 0.5, green: 0, blue: 1, white: 0 },
  { name: 'Pink', red: 1, green: 0.4, blue: 0.7, white: 0 },
  { name: 'Teal', red: 0, green: 0.5, blue: 0.5, white: 0 },
  { name: 'Lime', red: 0.5, green: 1, blue: 0, white: 0 },
  { name: 'Indigo', red: 0.3, green: 0, blue: 0.5, white: 0 },
  { name: 'Violet', red: 0.6, green: 0, blue: 1, white: 0 },
  { name: 'Coral', red: 1, green: 0.5, blue: 0.3, white: 0 },
  { name: 'Peach', red: 1, green: 0.8, blue: 0.6, white: 0 },
  { name: 'Mint', red: 0.6, green: 1, blue: 0.8, white: 0 },
  { name: 'Lavender', red: 0.9, green: 0.6, blue: 1, white: 0 },
  { name: 'Salmon', red: 1, green: 0.6, blue: 0.5, white: 0 },
  { name: 'Gold', red: 1, green: 0.84, blue: 0, white: 0 },
  { name: 'Sky Blue', red: 0.5, green: 0.8, blue: 1, white: 0 },
  { name: 'Hot Pink', red: 1, green: 0.2, blue: 0.6, white: 0 },
  { name: 'Turquoise', red: 0.2, green: 0.9, blue: 0.8, white: 0 },
];

export default function SpecificColorsPage() {
  const [brightness, setBrightness] = useState(1.0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);

  const handleColorSelect = async (color: PresetColor) => {
    setLoading(true);
    setError(null);

    try {
      // Apply brightness scaling to all color channels
      const config: ManagerConfig = {
        behaviors: [
          {
            behavior_type: 'fixed',
            color: 'red',
            config: {
              power_value: color.red * brightness,
            },
          },
          {
            behavior_type: 'fixed',
            color: 'green',
            config: {
              power_value: color.green * brightness,
            },
          },
          {
            behavior_type: 'fixed',
            color: 'blue',
            config: {
              power_value: color.blue * brightness,
            },
          },
          {
            behavior_type: 'fixed',
            color: 'white',
            config: {
              power_value: color.white * brightness,
            },
          },
        ],
      };

      await registerBehavior(config);
      await turnLightsOn();
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : 'Failed to load color';
      setError(message);
      setSnackbarOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const handleBrightnessChange = (_: Event, value: number | number[]) => {
    setBrightness(Array.isArray(value) ? value[0] : value);
  };

  const handleSnackbarClose = () => {
    setSnackbarOpen(false);
  };

  return (
    <>
      <Container maxWidth="lg">
        <Box
          sx={{
            mt: 4,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 3,
          }}
        >
          <Typography variant="h4" component="h1" gutterBottom>
            Specific Colors
          </Typography>

          <Box sx={{ width: '100%', maxWidth: 400, px: 2 }}>
            <Typography variant="body2" gutterBottom>
              Brightness
            </Typography>
            <Slider
              value={brightness}
              onChange={handleBrightnessChange}
              min={0}
              max={1}
              step={0.01}
            />
            <Typography variant="caption">
              {brightness.toFixed(2)}
            </Typography>
          </Box>

          <Box
            sx={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: 3,
              justifyContent: 'center',
              maxWidth: 900,
            }}
          >
            {PRESET_COLORS.map((color) => (
              <PresetColorCard
                key={color.name}
                color={color}
                brightness={brightness}
                onClick={() => handleColorSelect(color)}
                disabled={loading}
              />
            ))}
          </Box>
        </Box>
      </Container>

      <Backdrop
        sx={{ color: '#fff', zIndex: (theme) => theme.zIndex.drawer + 1 }}
        open={loading}
      >
        <CircularProgress color="inherit" />
      </Backdrop>

      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={handleSnackbarClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert
          onClose={handleSnackbarClose}
          severity="error"
          sx={{ width: '100%' }}
        >
          {error}
        </Alert>
      </Snackbar>
    </>
  );
}
