import { Container, Typography, Box } from '@mui/material';

export default function DashboardPage() {
  return (
    <Container maxWidth="lg">
      <Box sx={{ mt: 4 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          LED Control Dashboard
        </Typography>
      </Box>
    </Container>
  );
}
