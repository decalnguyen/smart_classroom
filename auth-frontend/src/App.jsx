import React from "react";
import Login from "./components/Login.js";
import SignUp from "./components/SignUp.js";
import Calendar from "./components/calendar/calendar.jsx";
import "./App.css";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { ColorModeContext, useMode } from "./theme.js";
import { CssBaseline, ThemeProvider } from "@mui/material";
import Topbar from "./components/global/Topbar.jsx";
import Sidebar from "./components/global/Sidebar.jsx";
import Dashboard from "./components/dashboard/dashboard.jsx";
import Student from "./components/student/student.jsx";
import Attandance from "./components/attandance/attandance.jsx";
import BarChart from "./components/charts/BarChart.jsx";
import PieChart from "./components/charts/PieChart.jsx";
import LineChart from "./components/charts/LineChart.jsx";
import { useAuth, AuthProvider } from "./AuthContext.jsx";
import { PrivateRoutes, PublicRoutes } from "./Routes.js";
import { useLocation } from "react-router-dom";
import CalendarComponent from "./components/Calendar.js";

function AppContent() {
  const location = useLocation();
  const hideSidebar = location.pathname === "/login" || location.pathname === "/signup";
  const [theme, colorMode] = useMode();
  return (
    <AuthProvider>
      <ColorModeContext.Provider value={colorMode}>
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <div className="app">
            {!hideSidebar && <Sidebar />}
            <main className="content">
              {!hideSidebar && <Topbar />}
              <Routes>
                {/* Public Routes */}
                <Route path="/login" element={<PublicRoutes element={<Login />} />} />
                <Route path="/signup" element={<PublicRoutes element={<SignUp />} />} /> 

                {/* Private Routes */}
                <Route path="/dashboard" element={<PrivateRoutes element={<Dashboard />} />} />
                <Route path="/students" element={<PrivateRoutes element={<Student />} />} />
                <Route path="/attandance" element={<PrivateRoutes element={<Attandance />} />} />
                <Route path="/calendar" element={<PrivateRoutes element={<Calendar />} />} />
                <Route path="/calendarComp" element={<PrivateRoutes element={<CalendarComponent />} />} />
                <Route path="/barchart" element={<PrivateRoutes element={<BarChart />} />} />
                <Route path="/piechart" element={<PrivateRoutes element={<PieChart />} />} /> 
                <Route path="/linechart" element={<PrivateRoutes element={<LineChart />} />} /> 
              </Routes>
            </main>
          </div>
        </ThemeProvider>
      </ColorModeContext.Provider>
    </AuthProvider>
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