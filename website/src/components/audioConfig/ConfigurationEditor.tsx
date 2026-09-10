import {
  Box,
  Button,
  Paper,
  Typography,
  TextField,
  Divider,
  Stack,
  IconButton,
  Grid,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import StarIcon from '@mui/icons-material/Star';
import { BandEditor } from './BandEditor';
import type { SavedAudioConfig, AudioTuningBandConfig } from '../../types/api';

interface ConfigurationEditorProps {
  config: SavedAudioConfig;
  isActive: boolean;
  hasChanges: boolean;
  onConfigChange: (config: SavedAudioConfig) => void;
  onSave: () => void;
  onDiscard: () => void;
  onSetActive: () => void;
  onDelete: () => void;
}

export function ConfigurationEditor({
  config,
  isActive,
  hasChanges,
  onConfigChange,
  onSave,
  onDiscard,
  onSetActive,
  onDelete,
}: ConfigurationEditorProps) {
  const updateBandConfig = (band: 'bass' | 'mid_low' | 'mid_high' | 'treble', bandConfig: AudioTuningBandConfig) => {
    onConfigChange({
      ...config,
      config: {
        ...config.config,
        [band]: bandConfig,
      },
    });
  };

  const canDelete = config.name !== 'default' && !isActive;

  return (
    <Paper sx={{ p: 2, mb: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Typography variant="h6">
          Editing: {config.display_name}
          {hasChanges && (
            <Typography component="span" color="warning.main" sx={{ ml: 1 }}>
              (unsaved changes)
            </Typography>
          )}
        </Typography>
        <Stack direction="row" spacing={1}>
          {!isActive && (
            <Button
              startIcon={<StarIcon />}
              variant="outlined"
              onClick={onSetActive}
            >
              Set as Active
            </Button>
          )}
          {canDelete && (
            <IconButton color="error" onClick={onDelete}>
              <DeleteIcon />
            </IconButton>
          )}
        </Stack>
      </Stack>

      <Divider sx={{ mb: 2 }} />

      {/* Metadata */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Display Name"
            value={config.display_name}
            onChange={(e) => onConfigChange({ ...config, display_name: e.target.value })}
            fullWidth
            size="small"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Description"
            value={config.description}
            onChange={(e) => onConfigChange({ ...config, description: e.target.value })}
            fullWidth
            size="small"
          />
        </Grid>
      </Grid>

      {/* Frequency Cutoffs */}
      <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mb: 1 }}>
        Frequency Band Cutoffs (Hz)
      </Typography>
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 4 }}>
          <TextField
            label="Bass Cutoff"
            type="number"
            value={config.config.bass_cutoff}
            onChange={(e) => onConfigChange({
              ...config,
              config: { ...config.config, bass_cutoff: parseFloat(e.target.value) || 0 }
            })}
            fullWidth
            size="small"
            helperText={`0 - ${config.config.bass_cutoff} Hz`}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
          <TextField
            label="Mid-High Cutoff"
            type="number"
            value={config.config.mid_high_cutoff}
            onChange={(e) => onConfigChange({
              ...config,
              config: { ...config.config, mid_high_cutoff: parseFloat(e.target.value) || 0 }
            })}
            fullWidth
            size="small"
            helperText={`${config.config.bass_cutoff} - ${config.config.mid_high_cutoff} Hz`}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
          <TextField
            label="Treble Cutoff"
            type="number"
            value={config.config.treble_cutoff}
            onChange={(e) => onConfigChange({
              ...config,
              config: { ...config.config, treble_cutoff: parseFloat(e.target.value) || 0 }
            })}
            fullWidth
            size="small"
            helperText={`${config.config.mid_high_cutoff}+ Hz`}
          />
        </Grid>
      </Grid>

      {/* Band Settings */}
      <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mb: 2 }}>
        Band Settings
      </Typography>
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <BandEditor
            label="Bass"
            bandConfig={config.config.bass}
            onChange={(bandConfig) => updateBandConfig('bass', bandConfig)}
            color="#ff4444"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <BandEditor
            label="Mid-Low"
            bandConfig={config.config.mid_low}
            onChange={(bandConfig) => updateBandConfig('mid_low', bandConfig)}
            color="#44ff44"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <BandEditor
            label="Mid-High"
            bandConfig={config.config.mid_high}
            onChange={(bandConfig) => updateBandConfig('mid_high', bandConfig)}
            color="#4444ff"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <BandEditor
            label="Treble"
            bandConfig={config.config.treble}
            onChange={(bandConfig) => updateBandConfig('treble', bandConfig)}
            color="#ff44ff"
          />
        </Grid>
      </Grid>

      {/* Action Buttons */}
      <Box sx={{ display: 'flex', gap: 2 }}>
        <Button
          variant="contained"
          color="primary"
          onClick={onSave}
          disabled={!hasChanges}
        >
          Save Changes
        </Button>
        <Button
          variant="outlined"
          onClick={onDiscard}
          disabled={!hasChanges}
        >
          Discard Changes
        </Button>
      </Box>
    </Paper>
  );
}
