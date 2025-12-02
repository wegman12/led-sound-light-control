import { Box, Slider, TextField, Typography, Paper } from '@mui/material';

interface ColorPickerProps {
  red: number;
  green: number;
  blue: number;
  white: number;
  onRedChange: (value: number) => void;
  onGreenChange: (value: number) => void;
  onBlueChange: (value: number) => void;
  onWhiteChange: (value: number) => void;
}

export default function ColorPicker({
  red,
  green,
  blue,
  white,
  onRedChange,
  onGreenChange,
  onBlueChange,
  onWhiteChange,
}: ColorPickerProps) {
  const handleSliderChange = (setter: (value: number) => void) => (
    _: Event,
    value: number | number[]
  ) => {
    setter(Array.isArray(value) ? value[0] : value);
  };

  const handleInputChange = (setter: (value: number) => void) => (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const value = parseFloat(event.target.value);
    if (!isNaN(value) && value >= 0 && value <= 1) {
      setter(value);
    }
  };

  // Calculate RGB preview (mix RGB with white)
  const previewRed = Math.round((red + white) * 255);
  const previewGreen = Math.round((green + white) * 255);
  const previewBlue = Math.round((blue + white) * 255);
  const previewColor = `rgb(${Math.min(previewRed, 255)}, ${Math.min(previewGreen, 255)}, ${Math.min(previewBlue, 255)})`;

  return (
    <Box sx={{ width: '100%', maxWidth: 600 }}>
      {/* Color Preview */}
      <Paper
        elevation={3}
        sx={{
          width: '100%',
          height: 150,
          mb: 4,
          backgroundColor: previewColor,
          border: '2px solid',
          borderColor: 'divider',
        }}
      />

      {/* Red Channel */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="body2" color="error" gutterBottom>
          Red
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Slider
            value={red}
            onChange={handleSliderChange(onRedChange)}
            min={0}
            max={1}
            step={0.01}
            sx={{ flex: 1 }}
            color="error"
          />
          <TextField
            value={red.toFixed(2)}
            onChange={handleInputChange(onRedChange)}
            type="number"
            inputProps={{ min: 0, max: 1, step: 0.01 }}
            sx={{ width: 100 }}
            size="small"
          />
        </Box>
      </Box>

      {/* Green Channel */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="body2" color="success.main" gutterBottom>
          Green
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Slider
            value={green}
            onChange={handleSliderChange(onGreenChange)}
            min={0}
            max={1}
            step={0.01}
            sx={{ flex: 1 }}
            color="success"
          />
          <TextField
            value={green.toFixed(2)}
            onChange={handleInputChange(onGreenChange)}
            type="number"
            inputProps={{ min: 0, max: 1, step: 0.01 }}
            sx={{ width: 100 }}
            size="small"
          />
        </Box>
      </Box>

      {/* Blue Channel */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="body2" color="primary" gutterBottom>
          Blue
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Slider
            value={blue}
            onChange={handleSliderChange(onBlueChange)}
            min={0}
            max={1}
            step={0.01}
            sx={{ flex: 1 }}
            color="primary"
          />
          <TextField
            value={blue.toFixed(2)}
            onChange={handleInputChange(onBlueChange)}
            type="number"
            inputProps={{ min: 0, max: 1, step: 0.01 }}
            sx={{ width: 100 }}
            size="small"
          />
        </Box>
      </Box>

      {/* White Channel */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="body2" gutterBottom>
          White
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Slider
            value={white}
            onChange={handleSliderChange(onWhiteChange)}
            min={0}
            max={1}
            step={0.01}
            sx={{ flex: 1 }}
          />
          <TextField
            value={white.toFixed(2)}
            onChange={handleInputChange(onWhiteChange)}
            type="number"
            inputProps={{ min: 0, max: 1, step: 0.01 }}
            sx={{ width: 100 }}
            size="small"
          />
        </Box>
      </Box>
    </Box>
  );
}
