import React from "react";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import CalendarComponent from "./components/Calendar";
import "./App.css";
import { BrowserRouter, Routes, Route, useLocation } from "react-router-dom";
import { ColorModeContext, useMode } from "./theme";  
import { CssBaseline, ThemeProvider } from "@mui/material";
import Topbar from "./components/global/Topbar";  
import Sidebar from "./components/global/Sidebar";
import Dashboard from "./components/dashboard/dashboard";
import Student from "./components/student/student";
// import Contacts from "./components/contacts";
// import Invoices from "./components/invoices";
// import Form from "./components/form";
// import FAQ from "./components/faq";
// import Bar from "./components/bar";
// import Pie from "./components/pie";
// import Line from "./components/line";

function AppContent() {
  const location = useLocation(); // Now inside BrowserRouter context
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
              <Route path="/dashboard" exact element={<Dashboard />} />
              <Route path="/students" element={<Student />} />
              <Route path="/login" element={<Login />} />
              <Route path="/signup" element={<SignUp />} />
              <Route path="/calendar" element={<CalendarComponent />} />
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
      <AppContent />
    </BrowserRouter>
  );
}

export default App;