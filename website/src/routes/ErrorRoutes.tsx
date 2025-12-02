import {Route, Navigate, Routes} from 'react-router-dom';
import ApiErrorPage from '../pages/ApiErrorPage';

export function ErrorRoutes() {
  return (
    <Routes>
      <Route path="/api-error" element={<ApiErrorPage />} />
      <Route path="*" element={<Navigate to="/api-error" replace />} />
    </Routes>
  );
}
