import { DashboardLayout } from "./components/DashboardLayout";
import { Toaster } from "react-hot-toast";
import { Routes, Route, Outlet } from "react-router-dom";
import { DashboardPage } from "./pages/DashboardPage";
import { LandingPage } from "./pages/LandingPage";
import { AboutPage } from './pages/AboutPage';
import UploadsPage from './pages/UploadsPage';
import ClaimsPage from './pages/ClaimsPage.tsx';

function App() {
  const handleUploadSuccess = () => {};

  const AppLayout = () => (
    <DashboardLayout onUploadSuccess={handleUploadSuccess}>
      <Outlet />
    </DashboardLayout>
  );

  return (
    <>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route element={<AppLayout />}>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/about" element={<AboutPage />} />
          <Route path="/uploads" element={<UploadsPage />} />
          <Route path="/claims" element={<ClaimsPage />} />
        </Route>
      </Routes>
      <Toaster />
    </>
  );
}

export default App;
