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
} from '@mui/material';
import ColorPicker from '../components/ColorPicker';
import { registerBehavior, turnLightsOn, ApiError } from '../services';
import type { ManagerConfig } from '../types/api';

export default function ColorPickerPage() {
  const [red, setRed] = useState(0.5);
  const [green, setGreen] = useState(0.5);
  const [blue, setBlue] = useState(0.5);
  const [white, setWhite] = useState(0.5);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);

  const handleLoadColor = async () => {
    setLoading(true);
    setError(null);

    try {
      // Create behavior configuration for RGBW
      const config: ManagerConfig = {
        behaviors: [
          {
            behavior_type: 'fixed',
            color: 'red',
            config: {
              power_value: red,
            },
          },
          {
            behavior_type: 'fixed',
            color: 'green',
            config: {
              power_value: green,
            },
          },
          {
            behavior_type: 'fixed',
            color: 'blue',
            config: {
              power_value: blue,
            },
          },
          {
            behavior_type: 'fixed',
            color: 'white',
            config: {
              power_value: white,
            },
          },
        ],
      };

      // Register the behavior
      await registerBehavior(config);

      // Turn on the lights
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
            Color Picker
          </Typography>
          <Typography variant="body1" color="text.secondary" gutterBottom>
            Select a static color for the LEDs
          </Typography>

          <ColorPicker
            red={red}
            green={green}
            blue={blue}
            white={white}
            onRedChange={setRed}
            onGreenChange={setGreen}
            onBlueChange={setBlue}
            onWhiteChange={setWhite}
          />

          <Button
            variant="contained"
            size="large"
            onClick={handleLoadColor}
            disabled={loading}
            sx={{ mt: 2 }}
          >
            Load Color
          </Button>
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
