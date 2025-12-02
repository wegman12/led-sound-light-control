import { Alert, Container, Typography, Box } from '@mui/material';
import WarningIcon from '@mui/icons-material/Warning';

export default function ApiErrorPage() {
  return (
    <Container maxWidth="md">
      <Box sx={{ mt: 4 }}>
        <Alert severity="error" icon={<WarningIcon />}>
          <Typography variant="h6" gutterBottom>
            API Connection Error
          </Typography>
          <Typography variant="body1">
            The LED control API is not available. Please ensure the API server
            is running and accessible.
          </Typography>
        </Alert>
      </Box>
    </Container>
  );
}
