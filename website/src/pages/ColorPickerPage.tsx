import { Container, Typography, Box } from '@mui/material';

export default function ColorPickerPage() {
  return (
    <Container maxWidth="lg">
      <Box
        sx={{
          mt: 4,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
        }}
      >
        <Typography variant="h4" component="h1" gutterBottom>
          Color Picker
        </Typography>
        <Typography variant="body1" color="text.secondary">
          Select a static color for the LEDs
        </Typography>
      </Box>
    </Container>
  );
}
