import { Box, Slider, Typography } from '@mui/material';
import Wheel from '@uiw/react-color-wheel';
import { hsvaToHsla, type HsvaColor } from '@uiw/color-convert';

interface ColorPickerProps {
  hsva: HsvaColor;
  white: number;
  onHsvaChange: (hsva: HsvaColor) => void;
  onWhiteChange: (value: number) => void;
}

export default function ColorPicker({
  hsva,
  white,
  onHsvaChange,
  onWhiteChange,
}: ColorPickerProps) {
  const handleWhiteSliderChange = (_: Event, value: number | number[]) => {
    onWhiteChange(Array.isArray(value) ? value[0] : value);
  };

  // Handle wheel change (hue and saturation)
  const handleWheelChange = (color: { hsva: HsvaColor }) => {
    // Keep the current value (brightness), only update hue and saturation
    onHsvaChange({ h: color.hsva.h, s: color.hsva.s, v: hsva.v, a: 1 });
  };

  // Handle brightness slider change
  const handleBrightnessChange = (_: Event, value: number | number[]) => {
    const brightness = Array.isArray(value) ? value[0] : value;
    onHsvaChange({ ...hsva, v: brightness });
  };

  // Calculate brightness slider color gradient
  const minBrightnessColor = hsvaToHsla({ ...hsva, v: 0 });
  const maxBrightnessColor = hsvaToHsla({ ...hsva, v: 100 });

  return (
    <Box sx={{ width: '100%', maxWidth: 700 }}>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 3,
        }}
      >
        {/* RGB Brightness Slider */}
        <Box sx={{ width: '100%', maxWidth: 400, px: 2 }}>
          <Typography variant="body2" gutterBottom>
            RGB Brightness
          </Typography>
          <Slider
            value={hsva.v}
            onChange={handleBrightnessChange}
            min={0}
            max={100}
            step={1}
            sx={{
              '& .MuiSlider-track': {
                background: `linear-gradient(to right,
                  hsla(${minBrightnessColor.h}, ${minBrightnessColor.s}%, ${minBrightnessColor.l}%, 1),
                  hsla(${maxBrightnessColor.h}, ${maxBrightnessColor.s}%, ${maxBrightnessColor.l}%, 1))`,
                border: 'none',
              },
              '& .MuiSlider-rail': {
                background: `linear-gradient(to right,
                  hsla(${minBrightnessColor.h}, ${minBrightnessColor.s}%, ${minBrightnessColor.l}%, 1),
                  hsla(${maxBrightnessColor.h}, ${maxBrightnessColor.s}%, ${maxBrightnessColor.l}%, 1))`,
              },
            }}
          />
        </Box>

        {/* Color Wheel */}
        <Box>
          <Wheel color={hsva} onChange={handleWheelChange} width={280} height={280} />
        </Box>

        {/* White Channel Slider */}
        <Box sx={{ width: '100%', maxWidth: 400, px: 2 }}>
          <Typography variant="body2" gutterBottom>
            White
          </Typography>
          <Slider
            value={white}
            onChange={handleWhiteSliderChange}
            min={0}
            max={1}
            step={0.01}
          />
          <Typography variant="caption">
            {white.toFixed(2)}
          </Typography>
        </Box>
      </Box>
    </Box>
  );
}
