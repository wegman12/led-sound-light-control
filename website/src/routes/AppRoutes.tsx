import {Route, Navigate, Routes} from 'react-router-dom';
import DefaultLayout from '../layouts/DefaultLayout';
import DashboardPage from '../pages/DashboardPage';
import ColorPickerPage from '../pages/ColorPickerPage';

export function AppRoutes() {
  return (
      <Routes>
        <Route element={<DefaultLayout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/color-picker" element={<ColorPickerPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
  );
}
