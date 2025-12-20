import { useState } from 'react';
import {
  Box,
  Button,
  Paper,
  Typography,
  Stack,
  Chip,
  Collapse,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import StopIcon from '@mui/icons-material/Stop';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { AudioStreamChart } from './AudioStreamChart';
import { useAudioStream } from '../../hooks/useAudioStream';

interface VisibleChannels {
  bass: boolean;
  midLow: boolean;
  midHigh: boolean;
  treble: boolean;
}

export function AudioStreamPreview() {
  const [expanded, setExpanded] = useState(false);
  const [viewMode, setViewMode] = useState<'raw' | 'processed' | 'both'>('processed');
  const [visibleChannels, setVisibleChannels] = useState<VisibleChannels>({
    bass: true,
    midLow: true,
    midHigh: true,
    treble: true,
  });

  const {
    chartData,
    isStreaming,
    isConnected,
    startStreaming,
    stopStreaming,
  } = useAudioStream();

  const handleToggleChannel = (channel: keyof VisibleChannels) => {
    setVisibleChannels((prev) => ({
      ...prev,
      [channel]: !prev[channel],
    }));
  };

  const handleToggleStreaming = async () => {
    try {
      if (isStreaming) {
        await stopStreaming();
      } else {
        await startStreaming();
        setExpanded(true);
      }
    } catch {
      // Error is handled in the hook
    }
  };

  return (
    <Paper sx={{ p: 2 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: expanded ? 2 : 0 }}>
        <Stack direction="row" alignItems="center" spacing={2}>
          <Typography variant="h6">Real-Time Audio Preview</Typography>
          <Chip
            label={isConnected ? 'Connected' : 'Disconnected'}
            color={isConnected ? 'success' : 'error'}
            size="small"
            variant="outlined"
          />
        </Stack>
        <Stack direction="row" spacing={1}>
          <Button
            variant={isStreaming ? 'contained' : 'outlined'}
            color={isStreaming ? 'error' : 'primary'}
            startIcon={isStreaming ? <StopIcon /> : <PlayArrowIcon />}
            onClick={handleToggleStreaming}
            disabled={!isConnected}
          >
            {isStreaming ? 'Stop' : 'Start'}
          </Button>
          <Button
            variant="text"
            onClick={() => setExpanded(!expanded)}
            endIcon={expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          >
            {expanded ? 'Hide' : 'Show'}
          </Button>
        </Stack>
      </Stack>

      <Collapse in={expanded}>
        <Box sx={{ mt: 2 }}>
          {chartData.length === 0 && !isStreaming ? (
            <Typography color="text.secondary" sx={{ textAlign: 'center', py: 4 }}>
              Click "Start" to begin streaming audio data
            </Typography>
          ) : (
            <AudioStreamChart
              chartData={chartData}
              viewMode={viewMode}
              visibleChannels={visibleChannels}
              onViewModeChange={setViewMode}
              onToggleChannel={handleToggleChannel}
            />
          )}
        </Box>
      </Collapse>
    </Paper>
  );
}
