import { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Typography,
  TextField,
  Alert,
  Snackbar,
  Stack,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  listConfigs,
  getConfig,
  createConfig,
  updateConfig,
  deleteConfig,
  setActiveConfig,
} from '../services';
import { ConfigurationList, ConfigurationEditor } from '../components/audioConfig';
import type {
  SavedAudioConfigSummary,
  SavedAudioConfig,
  AudioTuningConfig,
} from '../types/api';

const getDefaultTuningConfig = (): AudioTuningConfig => ({
  bass_cutoff: 100,
  mid_high_cutoff: 500,
  treble_cutoff: 2000,
  bass: {
    scaling_factor: 0.000001,
    noise_threshold: 92565436,
    min_power_value: 0.0,
    max_power_value: 1.0,
    smoothing: 0.3,
  },
  mid_low: {
    scaling_factor: 0.000001,
    noise_threshold: 18541891,
    min_power_value: 0.0,
    max_power_value: 1.0,
    smoothing: 0.3,
  },
  mid_high: {
    scaling_factor: 0.000002,
    noise_threshold: 10913229,
    min_power_value: 0.0,
    max_power_value: 1.0,
    smoothing: 0.3,
  },
  treble: {
    scaling_factor: 0.000027,
    noise_threshold: 2258769,
    min_power_value: 0.0,
    max_power_value: 1.0,
    smoothing: 0.3,
  },
});

export function AudioConfigurationPage() {
  const [configs, setConfigs] = useState<SavedAudioConfigSummary[]>([]);
  const [activeConfigName, setActiveConfigName] = useState<string>('');
  const [selectedConfig, setSelectedConfig] = useState<SavedAudioConfig | null>(null);
  const [editedConfig, setEditedConfig] = useState<SavedAudioConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Dialog states
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [newConfigName, setNewConfigName] = useState('');
  const [newConfigDisplayName, setNewConfigDisplayName] = useState('');
  const [newConfigDescription, setNewConfigDescription] = useState('');

  useEffect(() => {
    loadConfigs();
  }, []);

  const loadConfigs = async () => {
    try {
      setLoading(true);
      const response = await listConfigs();
      setConfigs(response.configs);
      setActiveConfigName(response.active_config);

      if (response.active_config) {
        const config = await getConfig(response.active_config);
        setSelectedConfig(config);
        setEditedConfig(JSON.parse(JSON.stringify(config)));
      }
    } catch (err) {
      setError('Failed to load configurations: ' + (err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectConfig = async (name: string) => {
    try {
      const config = await getConfig(name);
      setSelectedConfig(config);
      setEditedConfig(JSON.parse(JSON.stringify(config)));
    } catch (err) {
      setError('Failed to load configuration: ' + (err as Error).message);
    }
  };

  const handleSaveChanges = async () => {
    if (!editedConfig || !selectedConfig) return;

    try {
      await updateConfig(selectedConfig.name, {
        display_name: editedConfig.display_name,
        description: editedConfig.description,
        config: editedConfig.config,
      });
      setSuccess('Configuration saved successfully');
      await loadConfigs();
    } catch (err) {
      setError('Failed to save configuration: ' + (err as Error).message);
    }
  };

  const handleSetActive = async () => {
    if (!selectedConfig) return;

    try {
      await setActiveConfig(selectedConfig.name);
      setActiveConfigName(selectedConfig.name);
      setSuccess(`"${selectedConfig.display_name}" is now the active configuration`);
      await loadConfigs();
    } catch (err) {
      setError('Failed to set active configuration: ' + (err as Error).message);
    }
  };

  const handleDiscardChanges = () => {
    if (selectedConfig) {
      setEditedConfig(JSON.parse(JSON.stringify(selectedConfig)));
    }
  };

  const handleCreateConfig = async () => {
    if (!newConfigName.trim()) {
      setError('Configuration name is required');
      return;
    }

    try {
      await createConfig({
        name: newConfigName.toLowerCase().replace(/\s+/g, '-'),
        display_name: newConfigDisplayName || newConfigName,
        description: newConfigDescription,
        config: getDefaultTuningConfig(),
      });
      setSuccess('Configuration created successfully');
      setCreateDialogOpen(false);
      setNewConfigName('');
      setNewConfigDisplayName('');
      setNewConfigDescription('');
      await loadConfigs();
    } catch (err) {
      setError('Failed to create configuration: ' + (err as Error).message);
    }
  };

  const handleDeleteConfig = async () => {
    if (!selectedConfig) return;

    try {
      await deleteConfig(selectedConfig.name);
      setSuccess('Configuration deleted successfully');
      setDeleteDialogOpen(false);
      setSelectedConfig(null);
      setEditedConfig(null);
      await loadConfigs();
    } catch (err) {
      setError('Failed to delete configuration: ' + (err as Error).message);
    }
  };

  const hasChanges = editedConfig && selectedConfig &&
    JSON.stringify(editedConfig) !== JSON.stringify(selectedConfig);

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography>Loading configurations...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Audio Configuration
      </Typography>

      <ConfigurationList
        configs={configs}
        activeConfigName={activeConfigName}
        selectedConfigName={selectedConfig?.name || null}
        onSelectConfig={handleSelectConfig}
        onCreateNew={() => setCreateDialogOpen(true)}
      />

      {editedConfig && (
        <ConfigurationEditor
          config={editedConfig}
          isActive={selectedConfig?.name === activeConfigName}
          hasChanges={!!hasChanges}
          onConfigChange={setEditedConfig}
          onSave={handleSaveChanges}
          onDiscard={handleDiscardChanges}
          onSetActive={handleSetActive}
          onDelete={() => setDeleteDialogOpen(true)}
        />
      )}

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)}>
        <DialogTitle>Create New Configuration</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1, minWidth: 300 }}>
            <TextField
              label="Name (identifier)"
              value={newConfigName}
              onChange={(e) => setNewConfigName(e.target.value)}
              fullWidth
              size="small"
              helperText="Letters, numbers, hyphens only"
            />
            <TextField
              label="Display Name"
              value={newConfigDisplayName}
              onChange={(e) => setNewConfigDisplayName(e.target.value)}
              fullWidth
              size="small"
            />
            <TextField
              label="Description"
              value={newConfigDescription}
              onChange={(e) => setNewConfigDescription(e.target.value)}
              fullWidth
              size="small"
              multiline
              rows={2}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleCreateConfig} variant="contained">Create</Button>
        </DialogActions>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>Delete Configuration</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete "{selectedConfig?.display_name}"?
          </Typography>
          <Typography color="error" sx={{ mt: 1 }}>
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleDeleteConfig} color="error" variant="contained">Delete</Button>
        </DialogActions>
      </Dialog>

      {/* Notifications */}
      <Snackbar
        open={!!error}
        autoHideDuration={6000}
        onClose={() => setError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setError(null)} severity="error">
          {error}
        </Alert>
      </Snackbar>

      <Snackbar
        open={!!success}
        autoHideDuration={4000}
        onClose={() => setSuccess(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setSuccess(null)} severity="success">
          {success}
        </Alert>
      </Snackbar>
    </Box>
  );
}
