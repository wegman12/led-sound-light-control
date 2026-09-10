import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  AppBar,
  Toolbar,
  Typography,
  Button,
  Box,
  Snackbar,
  Alert,
  IconButton,
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
} from '@mui/material';
import SettingsIcon from '@mui/icons-material/Settings';
import TuneIcon from '@mui/icons-material/Tune';
import { turnLightsOn, turnLightsOff, ApiError } from '../services';

export default function Header() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [settingsAnchorEl, setSettingsAnchorEl] = useState<null | HTMLElement>(null);
  const settingsMenuOpen = Boolean(settingsAnchorEl);

  const handleSettingsClick = (event: React.MouseEvent<HTMLElement>) => {
    setSettingsAnchorEl(event.currentTarget);
  };

  const handleSettingsClose = () => {
    setSettingsAnchorEl(null);
  };

  const handleNavigateToAudioConfig = () => {
    handleSettingsClose();
    navigate('/audio-configuration');
  };

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
          <Typography
            variant="h6"
            component="div"
            onClick={() => navigate('/')}
            sx={{ flexGrow: 1, cursor: 'pointer' }}
          >
            LED Manager
          </Typography>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
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
            <IconButton
              color="inherit"
              onClick={handleSettingsClick}
              aria-controls={settingsMenuOpen ? 'settings-menu' : undefined}
              aria-haspopup="true"
              aria-expanded={settingsMenuOpen ? 'true' : undefined}
            >
              <SettingsIcon />
            </IconButton>
            <Menu
              id="settings-menu"
              anchorEl={settingsAnchorEl}
              open={settingsMenuOpen}
              onClose={handleSettingsClose}
              anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'right',
              }}
              transformOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
            >
              <MenuItem onClick={handleNavigateToAudioConfig}>
                <ListItemIcon>
                  <TuneIcon fontSize="small" />
                </ListItemIcon>
                <ListItemText>Audio Configuration</ListItemText>
              </MenuItem>
            </Menu>
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
