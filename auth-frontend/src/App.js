import React from "react";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import Calendar from "./components/calendar/calendar";
import "./App.css";
import { BrowserRouter, Routes, Route } from "react-router-dom";
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
import { AuthProvider } from "./context/AuthContext";
import { PrivateRoute, PublicRoute } from "./components/Routes";

function AppContent() {
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
              <Route path="/login" element={<PublicRoute element={<Login />} />} />
              <Route path="/signup" element={<PublicRoute element={<SignUp />} />} />

              {/* Private Routes */}
              <Route path="/dashboard" element={<PrivateRoute element={<Dashboard />} />} />
              <Route path="/students" element={<PrivateRoute element={<Student />} />} />
              <Route path="/attandance" element={<PrivateRoute element={<Attandance />} />} />
              <Route path="/calendar" element={<PrivateRoute element={<Calendar />} />} />
              <Route path="/barchart" element={<PrivateRoute element={<BarChart />} />} />
              <Route path="/piechart" element={<PrivateRoute element={<PieChart />} />} />
              <Route path="/linechart" element={<PrivateRoute element={<LineChart />} />} />
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