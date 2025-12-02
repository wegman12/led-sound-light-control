import { BrowserRouter } from 'react-router-dom';
import { CircularProgress, Box } from '@mui/material';
import { useHealthCheck } from './hooks/useHealthCheck';
import { AppRoutes, ErrorRoutes } from './routes';

function App() {
  const { isHealthy, isLoading } = useHealthCheck();

  if (isLoading) {
    return (
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: '100vh',
        }}
      >
        <CircularProgress />
      </Box>
    );
  }

  return (
    <BrowserRouter>
        {isHealthy ? <AppRoutes /> : <ErrorRoutes />}
    </BrowserRouter>
  );
}

export default App;
