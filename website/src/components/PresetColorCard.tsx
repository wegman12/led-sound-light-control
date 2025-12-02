import { Card, CardActionArea, Typography, Box } from '@mui/material';

interface PresetColor {
  name: string;
  red: number;
  green: number;
  blue: number;
  white: number;
}

interface PresetColorCardProps {
  color: PresetColor;
  brightness: number;
  onClick: () => void;
  disabled?: boolean;
}

export default function PresetColorCard({
  color,
  brightness,
  onClick,
  disabled = false,
}: PresetColorCardProps) {
  // Calculate RGB display color (adding white to all channels) and apply brightness scaling
  const displayRed = Math.min(Math.round((color.red * brightness) * 255 + (color.white * brightness) * 255), 255);
  const displayGreen = Math.min(Math.round((color.green * brightness) * 255 + (color.white * brightness) * 255), 255);
  const displayBlue = Math.min(Math.round((color.blue * brightness) * 255 + (color.white * brightness) * 255), 255);
  const backgroundColor = `rgb(${displayRed}, ${displayGreen}, ${displayBlue})`;

  // Calculate text color based on luminance for better contrast
  const luminance = (displayRed * 299 + displayGreen * 587 + displayBlue * 114) / 1000;
  const textColor = luminance > 128 ? '#000000' : '#ffffff';

  return (
    <Card
      sx={{
        width: 150,
        height: 150,
        opacity: disabled ? 0.6 : 1,
      }}
    >
      <CardActionArea
        onClick={onClick}
        disabled={disabled}
        sx={{
          width: '100%',
          height: '100%',
          backgroundColor,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          '&:hover': {
            filter: 'brightness(0.9)',
          },
        }}
      >
        <Box
          sx={{
            textAlign: 'center',
            p: 2,
          }}
        >
          <Typography
            variant="h6"
            sx={{
              color: textColor,
              fontWeight: 'bold',
              textShadow: luminance > 128
                ? '1px 1px 2px rgba(0,0,0,0.2)'
                : '1px 1px 2px rgba(255,255,255,0.2)',
            }}
          >
            {color.name}
          </Typography>
        </Box>
      </CardActionArea>
    </Card>
  );
}
