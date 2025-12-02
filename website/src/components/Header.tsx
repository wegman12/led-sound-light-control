import { useState } from 'react';
import { AppBar, Toolbar, Typography, Button, Box, Snackbar, Alert } from '@mui/material';
import { turnLightsOn, turnLightsOff, ApiError } from '../services';

export default function Header() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);

  const handleTurnOn = async () => {
    setLoading(true);
    try {
      await turnLightsOn();
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Failed to turn lights on';
      setError(message);
      setSnackbarOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const handleTurnOff = async () => {
    setLoading(true);
    try {
      await turnLightsOff();
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Failed to turn lights off';
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
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            LED Manager
          </Typography>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button
              color="inherit"
              variant="outlined"
              onClick={handleTurnOn}
              disabled={loading}
            >
              On
            </Button>
            <Button
              color="inherit"
              variant="outlined"
              onClick={handleTurnOff}
              disabled={loading}
            >
              Off
            </Button>
          </Box>
        </Toolbar>
      </AppBar>
      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={handleSnackbarClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={handleSnackbarClose} severity="error" sx={{ width: '100%' }}>
          {error}
        </Alert>
      </Snackbar>
    </>
  );
}
