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

export default function FlashPage() {
  const [colorMode, setColorMode] = useState<ColorMode>('3');
  const [duration, setDuration] = useState(1000); // milliseconds
  const [brightness, setBrightness] = useState(1.0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);

  const handleLoadFlash = async () => {
    setLoading(true);
    setError(null);

    try {
      const durationStr = `${duration}ms`;

      // Create the joiner configuration based on color mode
      const config: ManagerConfig = colorMode === '3'
        ? create3ColorFlashConfig(durationStr, brightness)
        : create7ColorFlashConfig(durationStr, brightness);

      // Register the behavior
      await registerBehavior(config);

      // Turn on the lights
      await turnLightsOn();
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : 'Failed to load flash behavior';
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
            Flash
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
                Duration: {(duration / 1000).toFixed(2)}s
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
              onClick={handleLoadFlash}
              disabled={loading}
              fullWidth
              sx={{ mt: 2 }}
            >
              Load Flash
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
 * Creates a 3-color flash configuration (R, G, B alternating)
 */
function create3ColorFlashConfig(duration: string, brightness: number): ManagerConfig {
  const redJoiner: JoinerConfig = {
    behaviors: [
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: brightness },
      },
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
    ],
  };

  const greenJoiner: JoinerConfig = {
    behaviors: [
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: brightness },
      },
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
    ],
  };

  const blueJoiner: JoinerConfig = {
    behaviors: [
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: 0.0 },
      },
      {
        behavior_type: 'fixed',
        duration,
        behavior: { power_value: brightness },
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
 * Creates a 7-color flash configuration
 * Sequence: R, G, B, R+G (yellow), R+B (magenta), G+B (cyan), R+G+B (white)
 */
function create7ColorFlashConfig(duration: string, brightness: number): ManagerConfig {
  // Color sequence: R, G, B, RG, RB, GB, RGB
  const redValues = [brightness, 0.0, 0.0, brightness, brightness, 0.0, brightness];
  const greenValues = [0.0, brightness, 0.0, brightness, 0.0, brightness, brightness];
  const blueValues = [0.0, 0.0, brightness, 0.0, brightness, brightness, brightness];

  const redJoiner: JoinerConfig = {
    behaviors: redValues.map((value) => ({
      behavior_type: 'fixed',
      duration,
      behavior: { power_value: value },
    })),
  };

  const greenJoiner: JoinerConfig = {
    behaviors: greenValues.map((value) => ({
      behavior_type: 'fixed',
      duration,
      behavior: { power_value: value },
    })),
  };

  const blueJoiner: JoinerConfig = {
    behaviors: blueValues.map((value) => ({
      behavior_type: 'fixed',
      duration,
      behavior: { power_value: value },
    })),
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
