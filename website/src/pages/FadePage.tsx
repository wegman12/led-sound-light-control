import { useState } from 'react';
import {
  Container,
  Typography,
  Box,
  Button,
  Backdrop,
  CircularProgress,
  Snackbar,
  Alert,
  Slider,
  RadioGroup,
  FormControlLabel,
  Radio,
  FormControl,
  FormLabel,
  Paper,
} from '@mui/material';
import { registerBehavior, turnLightsOn, ApiError } from '../services';
import type { ManagerConfig, JoinerConfig } from '../types/api';

type ColorMode = '3' | '7';

export default function FadePage() {
  const [colorMode, setColorMode] = useState<ColorMode>('3');
  const [duration, setDuration] = useState(1000); // milliseconds per fade
  const [brightness, setBrightness] = useState(1.0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);

  const handleLoadFade = async () => {
    setLoading(true);
    setError(null);

    try {
      const durationStr = `${duration}ms`;

      // Create the fade configuration based on color mode
      const config: ManagerConfig = colorMode === '3'
        ? create3ColorFadeConfig(durationStr, brightness)
        : create7ColorFadeConfig(durationStr, brightness);

      // Register the behavior
      await registerBehavior(config);

      // Turn on the lights
      await turnLightsOn();
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : 'Failed to load fade behavior';
      setError(message);
      setSnackbarOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const handleSnackbarClose = () => {
    setSnackbarOpen(false);
  };

  const handleDurationChange = (_event: Event, value: number | number[]) => {
    setDuration(value as number);
  };

  const handleBrightnessChange = (_event: Event, value: number | number[]) => {
    setBrightness(value as number);
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
            Fade
          </Typography>

          <Paper elevation={2} sx={{ p: 3, width: '100%', maxWidth: 600 }}>
            <FormControl component="fieldset" sx={{ mb: 3 }}>
              <FormLabel component="legend">Color Mode</FormLabel>
              <RadioGroup
                value={colorMode}
                onChange={(e) => setColorMode(e.target.value as ColorMode)}
                row
              >
                <FormControlLabel
                  value="3"
                  control={<Radio />}
                  label="3 Colors (R, G, B)"
                />
                <FormControlLabel
                  value="7"
                  control={<Radio />}
                  label="7 Colors (All Combinations)"
                />
              </RadioGroup>
            </FormControl>

            <Box sx={{ mb: 3 }}>
              <Typography gutterBottom>
                Fade Duration: {(duration / 1000).toFixed(2)}s
              </Typography>
              <Slider
                value={duration}
                onChange={handleDurationChange}
                min={100}
                max={30000}
                step={100}
                valueLabelDisplay="auto"
                valueLabelFormat={(value) => `${(value / 1000).toFixed(2)}s`}
              />
            </Box>

            <Box sx={{ mb: 3 }}>
              <Typography gutterBottom>
                Brightness: {brightness.toFixed(2)}
              </Typography>
              <Slider
                value={brightness}
                onChange={handleBrightnessChange}
                min={0}
                max={1}
                step={0.01}
                valueLabelDisplay="auto"
                valueLabelFormat={(value) => value.toFixed(2)}
              />
            </Box>

            <Button
              variant="contained"
              size="large"
              onClick={handleLoadFade}
              disabled={loading}
              fullWidth
              sx={{ mt: 2 }}
            >
              Load Fade
            </Button>
          </Paper>
        </Box>
      </Container>

      {/* Loading Overlay */}
      <Backdrop
        sx={{ color: '#fff', zIndex: (theme) => theme.zIndex.drawer + 1 }}
        open={loading}
      >
        <CircularProgress color="inherit" />
      </Backdrop>

      {/* Error Snackbar */}
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

/**
 * Creates a 3-color fade configuration (R → G → B → R)
 * Uses skipper behavior to create smooth transitions between colors
 */
function create3ColorFadeConfig(duration: string, brightness: number): ManagerConfig {
  // Red: starts at brightness, fades to 0, stays at 0, fades back to brightness
  const redJoiner: JoinerConfig = {
    behaviors: [
      {
        behavior_type: 'skipper',
        duration,
        behavior: {
          duration,
          min_power_value: brightness,
          max_power_value: 0.0,
        },
      },
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
      {
        behavior_type: 'skipper',
        duration,
        behavior: {
          duration,
          min_power_value: 0.0,
          max_power_value: brightness,
        },
      },
    ],
  };

  // Green: stays at 0, fades to brightness, fades to 0
  const greenJoiner: JoinerConfig = {
    behaviors: [
      {
        behavior_type: 'skipper',
        duration,
        behavior: {
          duration,
          min_power_value: 0.0,
          max_power_value: brightness,
        },
      },
      {
        behavior_type: 'skipper',
        duration,
        behavior: {
          duration,
          min_power_value: brightness,
          max_power_value: 0.0,
        },
      },
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
    ],
  };

  // Blue: stays at 0, stays at 0, fades to brightness, then wraps to fade down
  // Note: The wrap-around is handled by using a second joiner cycle
  const blueJoiner: JoinerConfig = {
    behaviors: [
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
      {
        behavior_type: 'skipper',
        duration,
        behavior: {
          duration,
          min_power_value: 0.0,
          max_power_value: brightness,
        },
      },
      {
        behavior_type: 'skipper',
        duration,
        behavior: {
          duration,
          min_power_value: brightness,
          max_power_value: 0.0,
        },
      },
    ],
  };

  return {
    behaviors: [
      {
        behavior_type: 'joiner',
        color: 'red',
        config: redJoiner,
      },
      {
        behavior_type: 'joiner',
        color: 'green',
        config: greenJoiner,
      },
      {
        behavior_type: 'joiner',
        color: 'blue',
        config: blueJoiner,
      },
      {
        behavior_type: 'fixed',
        color: 'white',
        config: { power_value: 0.0 },
      },
    ],
  };
}

/**
 * Creates a 7-color fade configuration
 * Sequence: R → G → B → RG → RB → GB → RGB → R
 * Each transition fades smoothly between color combinations
 */
function create7ColorFadeConfig(duration: string, brightness: number): ManagerConfig {
  // Color sequence with transitions:
  // R(1,0,0) → G(0,1,0) → B(0,0,1) → RG(1,1,0) → RB(1,0,1) → GB(0,1,1) → RGB(1,1,1) → R(1,0,0)

  // For smooth transitions, we create fade patterns for each color channel
  // This is a complex sequence, so we'll create intermediate steps

  // Red channel: 1 → 0 → 0 → 1 → 1 → 0 → 1 → 1
  const redJoiner: JoinerConfig = {
    behaviors: [
      // R to G: fade down
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: brightness, max_power_value: 0.0 } },
      // G to B: stay low
      { behavior_type: 'fixed', duration, behavior: { power_value: 0.0 } },
      // B to RG: fade up
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: 0.0, max_power_value: brightness } },
      // RG to RB: stay high
      { behavior_type: 'fixed', duration, behavior: { power_value: brightness } },
      // RB to GB: fade down
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: brightness, max_power_value: 0.0 } },
      // GB to RGB: fade up
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: 0.0, max_power_value: brightness } },
      // RGB to R: stay high (wraps back to start)
      { behavior_type: 'fixed', duration, behavior: { power_value: brightness } },
    ],
  };

  // Green channel: 0 → 1 → 0 → 1 → 0 → 1 → 1
  const greenJoiner: JoinerConfig = {
    behaviors: [
      // R to G: fade up
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: 0.0, max_power_value: brightness } },
      // G to B: fade down
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: brightness, max_power_value: 0.0 } },
      // B to RG: fade up
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: 0.0, max_power_value: brightness } },
      // RG to RB: fade down
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: brightness, max_power_value: 0.0 } },
      // RB to GB: fade up
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: 0.0, max_power_value: brightness } },
      // GB to RGB: stay high
      { behavior_type: 'fixed', duration, behavior: { power_value: brightness } },
      // RGB to R: fade down
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: brightness, max_power_value: 0.0 } },
    ],
  };

  // Blue channel: 0 → 0 → 1 → 0 → 1 → 1 → 1
  const blueJoiner: JoinerConfig = {
    behaviors: [
      // R to G: stay low
      { behavior_type: 'fixed', duration, behavior: { power_value: 0.0 } },
      // G to B: fade up
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: 0.0, max_power_value: brightness } },
      // B to RG: fade down
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: brightness, max_power_value: 0.0 } },
      // RG to RB: fade up
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: 0.0, max_power_value: brightness } },
      // RB to GB: stay high
      { behavior_type: 'fixed', duration, behavior: { power_value: brightness } },
      // GB to RGB: stay high
      { behavior_type: 'fixed', duration, behavior: { power_value: brightness } },
      // RGB to R: fade down
      { behavior_type: 'skipper', duration, behavior: { duration, min_power_value: brightness, max_power_value: 0.0 } },
    ],
  };

  return {
    behaviors: [
      {
        behavior_type: 'joiner',
        color: 'red',
        config: redJoiner,
      },
      {
        behavior_type: 'joiner',
        color: 'green',
        config: greenJoiner,
      },
      {
        behavior_type: 'joiner',
        color: 'blue',
        config: blueJoiner,
      },
      {
        behavior_type: 'fixed',
        color: 'white',
        config: { power_value: 0.0 },
      },
    ],
  };
}
