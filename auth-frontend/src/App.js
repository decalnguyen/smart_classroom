import React from "react";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import Calendar from "./components/calendar/calendar";
import "./App.css";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { ColorModeContext, useMode } from "./theme";
import { CssBaseline, ThemeProvider } from "@mui/material";
import Topbar from "./components/global/Topbar";
import Sidebar from "./components/global/Sidebar";
import Dashboard from "./components/dashboard/dashboard";
import Student from "./components/student/student";
import Attandance from "./components/attandance/attandance";
import BarChart from "./components/charts/BarChart";
import PieChart from "./components/charts/PieChart";
import LineChart from "./components/charts/LineChart";
import { useAuth, AuthProvider } from "./AuthContext";
import { PrivateRoutes, PublicRoutes } from "./Routes";

function AppContent() {
  const {tokens} = useAuth();
  const ProtectedRoute = ({ element }) => {
    const { isAuthenticated } = useAuth();
    return isAuthenticated ? element : <Navigate to="/login" />;
  }
  const [theme, colorMode] = useMode();
  return (
    <ColorModeContext.Provider value={colorMode}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <div className="app">
          <Sidebar />
          <main className="content">
            <Topbar />
            <Routes>
              {/* Public Routes */}
              <Route path="/login" element={<PublicRoutes element={<Login />} />} />
              <Route path="/signup" element={<PublicRoutes element={<SignUp />} />} />

              {/* Private Routes */}
              <Route path="/dashboard" element={<PrivateRoutes element={<Dashboard />} />} />
              <Route path="/students" element={<PrivateRoutes element={<Student />} />} />
              <Route path="/attandance" element={<PrivateRoutes element={<Attandance />} />} />
              <Route path="/calendar" element={<PrivateRoutes element={<Calendar />} />} />
              <Route path="/barchart" element={<PrivateRoutes element={<BarChart />} />} />
              <Route path="/piechart" element={<PrivateRoutes element={<PieChart />} />} />
              <Route path="/linechart" element={<PrivateRoutes element={<LineChart />} />} />
            </Routes>
          </main>
        </div>
      </ThemeProvider>
    </ColorModeContext.Provider>
  );
}

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;