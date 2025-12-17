import {Route, Navigate, Routes} from 'react-router-dom';
import DefaultLayout from '../layouts/DefaultLayout';
import DashboardPage from '../pages/DashboardPage';
import ColorPickerPage from '../pages/ColorPickerPage';
import SpecificColorsPage from '../pages/SpecificColorsPage';
import FlashPage from '../pages/FlashPage';
import FadePage from '../pages/FadePage';
import ShimmerPage from '../pages/ShimmerPage';
import AudioLightsPage from '../pages/AudioLightsPage';
import AudioVisualizerPage from '../pages/AudioVisualizerPage';
import { AudioTuningPage } from '../pages/AudioTuningPage';

export function AppRoutes() {
  return (
      <Routes>
        <Route element={<DefaultLayout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/color-picker" element={<ColorPickerPage />} />
          <Route path="/specific-colors" element={<SpecificColorsPage />} />
          <Route path="/flash" element={<FlashPage />} />
          <Route path="/fade" element={<FadePage />} />
          <Route path="/shimmer" element={<ShimmerPage />} />
          <Route path="/audio-lights" element={<AudioLightsPage />} />
          <Route path="/audio-visualizer" element={<AudioVisualizerPage />} />
          <Route path="/audio-tuning" element={<AudioTuningPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
  );
}
