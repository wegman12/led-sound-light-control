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
import { hsvaToRgba, type HsvaColor } from '@uiw/color-convert';

export default function ColorPickerPage() {
  const [hsva, setHsva] = useState<HsvaColor>({ h: 120, s: 100, v: 100, a: 1 });
  const [white, setWhite] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);

  const handleLoadColor = async () => {
    setLoading(true);
    setError(null);

    try {
      // Convert HSVA to RGBA
      const rgba = hsvaToRgba(hsva);

      // Convert RGB values from 0-255 to 0.0-1.0 for API
      const redNormalized = rgba.r / 255;
      const greenNormalized = rgba.g / 255;
      const blueNormalized = rgba.b / 255;

      // Create behavior configuration for RGBW
      const config: ManagerConfig = {
        behaviors: [
          {
            behavior_type: 'fixed',
            color: 'red',
            config: {
              power_value: redNormalized,
            },
          },
          {
            behavior_type: 'fixed',
            color: 'green',
            config: {
              power_value: greenNormalized,
            },
          },
          {
            behavior_type: 'fixed',
            color: 'blue',
            config: {
              power_value: blueNormalized,
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

  // Calculate button color based on HSVA + white
  const rgba = hsvaToRgba(hsva);
  const buttonRed = Math.min(rgba.r + Math.round(white * 255), 255);
  const buttonGreen = Math.min(rgba.g + Math.round(white * 255), 255);
  const buttonBlue = Math.min(rgba.b + Math.round(white * 255), 255);
  const buttonColor = `rgb(${buttonRed}, ${buttonGreen}, ${buttonBlue})`;

  // Calculate text color based on brightness for better contrast
  const brightness = (buttonRed * 299 + buttonGreen * 587 + buttonBlue * 114) / 1000;
  const textColor = brightness > 128 ? '#000000' : '#ffffff';

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

          <ColorPicker
            hsva={hsva}
            white={white}
            onHsvaChange={setHsva}
            onWhiteChange={setWhite}
          />

          <Button
            variant="contained"
            size="large"
            onClick={handleLoadColor}
            disabled={loading}
            sx={{
              mt: 2,
              backgroundColor: buttonColor,
              color: textColor,
              '&:hover': {
                backgroundColor: buttonColor,
                filter: 'brightness(0.9)',
              },
              '&.Mui-disabled': {
                backgroundColor: buttonColor,
                color: textColor,
                opacity: 0.6,
              },
            }}
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
