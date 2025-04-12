import React from "react";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import Home from "./components/Home";
import Nav from "./components/Nav";
import Classrooms from "./components/classrooms";
import CalendarComponent from "./components/Calendar";
import "./App.css";
import { BrowserRouter, Routes, Route, useLocation } from "react-router-dom";
import { ColorModeContext, useMode } from "./theme";  
import { CssBaseline, ThemeProvider } from "@mui/material";
import Topbar from "./components/global/Topbar";  
import Sidebar from "./components/global/Sidebar";
import Dashboard from "./components/dashboard/dashboard";
function AppContent() {
  const location = useLocation(); // Now inside BrowserRouter context
  const [theme, colorMode] = useMode();
  return (
    <ColorModeContext.Provider value={colorMode}>
      <ThemeProvider theme={theme}> 
        <CssBaseline />
        {/* Add a button to toggle between light and dark mode */}
        <div className="app">
          {/* Conditionally render Nav based on the current path */}
          <main className="content">
            <Topbar />
            <Routes>
              <Route path="/" exact element={<Dashboard />} />
              <Route path="/login" element={<Login />} />
              <Route path="/signup" element={<SignUp />} />
              <Route path="/calendar" element={<CalendarComponent />} />
            </Routes>
            <Sidebar />
          </main>
        </div>
      </ThemeProvider>
    </ColorModeContext.Provider>
  );
}

function App() {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  );
}

export default App;