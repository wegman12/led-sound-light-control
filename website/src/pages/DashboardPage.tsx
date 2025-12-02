import { Container, Typography, Box } from '@mui/material';
import PaletteIcon from '@mui/icons-material/Palette';
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
        </Box>
      </Box>
    </Container>
  );
}
