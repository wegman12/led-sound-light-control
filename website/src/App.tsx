import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { CircularProgress, Box } from '@mui/material';
import { useHealthCheck } from './hooks/useHealthCheck';
import DashboardPage from './pages/DashboardPage';
import ApiErrorPage from './pages/ApiErrorPage';

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
      <Routes>
        {isHealthy ? (
          <>
            <Route path="/" element={<DashboardPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </>
        ) : (
          <>
            <Route path="/api-error" element={<ApiErrorPage />} />
            <Route path="*" element={<Navigate to="/api-error" replace />} />
          </>
        )}
      </Routes>
    </BrowserRouter>
  );
}

export default App;
