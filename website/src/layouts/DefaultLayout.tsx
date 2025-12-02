import { Box } from '@mui/material';
import { Outlet } from 'react-router-dom';
import Header from '../components/Header';

export default function DefaultLayout() {
  return (
    <Box sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Header />
      <Outlet />
    </Box>
  );
}
