import { Container, Typography, Box } from '@mui/material';
import PaletteIcon from '@mui/icons-material/Palette';
import GridViewIcon from '@mui/icons-material/GridView';
import BoltIcon from '@mui/icons-material/Bolt';
import GradientIcon from '@mui/icons-material/Gradient';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import MusicNoteIcon from '@mui/icons-material/MusicNote';
import BehaviorCard from '../components/BehaviorCard';

export default function DashboardPage() {
  return (
    <Container maxWidth="lg">
      <Box
        sx={{
          mt: 4,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 4,
        }}
      >
        <Typography variant="h4" component="h1" gutterBottom>
          LED Control Dashboard
        </Typography>
        <Box
          sx={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 3,
            justifyContent: 'center',
          }}
        >
          <BehaviorCard
            title="Color Picker"
            description="Select a static color for the LEDs"
            icon={<PaletteIcon sx={{ fontSize: 'inherit' }} />}
            path="/color-picker"
          />
          <BehaviorCard
            title="Specific Colors"
            description="Choose from preset colors"
            icon={<GridViewIcon sx={{ fontSize: 'inherit' }} />}
            path="/specific-colors"
          />
          <BehaviorCard
            title="Flash"
            description="Alternate between RGB color combinations"
            icon={<BoltIcon sx={{ fontSize: 'inherit' }} />}
            path="/flash"
          />
          <BehaviorCard
            title="Fade"
            description="Smoothly fade between colors"
            icon={<GradientIcon sx={{ fontSize: 'inherit' }} />}
            path="/fade"
          />
          <BehaviorCard
            title="Shimmer"
            description="Breathing effect with custom colors"
            icon={<AutoAwesomeIcon sx={{ fontSize: 'inherit' }} />}
            path="/shimmer"
          />
          <BehaviorCard
            title="Audio Lights"
            description="Audio-reactive lighting synchronized to music"
            icon={<MusicNoteIcon sx={{ fontSize: 'inherit' }} />}
            path="/audio-lights"
          />
        </Box>
      </Box>
    </Container>
  );
}
