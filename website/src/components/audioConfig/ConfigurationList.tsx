import {
  Box,
  Button,
  Paper,
  Typography,
  Stack,
  Radio,
  RadioGroup,
  FormControlLabel,
  Tooltip,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import StarIcon from '@mui/icons-material/Star';
import type { SavedAudioConfigSummary } from '../../types/api';

interface ConfigurationListProps {
  configs: SavedAudioConfigSummary[];
  activeConfigName: string;
  selectedConfigName: string | null;
  onSelectConfig: (name: string) => void;
  onCreateNew: () => void;
}

export function ConfigurationList({
  configs,
  activeConfigName,
  selectedConfigName,
  onSelectConfig,
  onCreateNew,
}: ConfigurationListProps) {
  return (
    <Paper sx={{ p: 2, mb: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Typography variant="h6">Saved Configurations</Typography>
        <Button
          startIcon={<AddIcon />}
          variant="outlined"
          onClick={onCreateNew}
        >
          New Configuration
        </Button>
      </Stack>

      <RadioGroup
        value={selectedConfigName || ''}
        onChange={(e) => onSelectConfig(e.target.value)}
      >
        <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', gap: 1 }}>
          {configs.map((config) => (
            <Paper
              key={config.name}
              sx={{
                p: 1.5,
                minWidth: 200,
                border: selectedConfigName === config.name ? '2px solid' : '1px solid',
                borderColor: selectedConfigName === config.name ? 'primary.main' : 'divider',
                cursor: 'pointer',
                transition: 'border-color 0.2s',
                '&:hover': {
                  borderColor: 'primary.light',
                },
              }}
              onClick={() => onSelectConfig(config.name)}
            >
              <Stack direction="row" alignItems="center" spacing={1}>
                <FormControlLabel
                  value={config.name}
                  control={<Radio size="small" />}
                  label=""
                  sx={{ m: 0 }}
                />
                <Box sx={{ flexGrow: 1 }}>
                  <Stack direction="row" alignItems="center" spacing={0.5}>
                    <Typography variant="subtitle2">{config.display_name}</Typography>
                    {config.name === activeConfigName && (
                      <Tooltip title="Active Configuration">
                        <StarIcon color="primary" sx={{ fontSize: 16 }} />
                      </Tooltip>
                    )}
                  </Stack>
                  <Typography variant="caption" color="text.secondary" noWrap sx={{ maxWidth: 180, display: 'block' }}>
                    {config.description || 'No description'}
                  </Typography>
                </Box>
              </Stack>
            </Paper>
          ))}
        </Stack>
      </RadioGroup>
    </Paper>
  );
}
