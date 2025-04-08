import React from "react";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import Home from "./components/Home";
import Nav from "./components/Nav";
import Classrooms from "./components/classrooms";
import "./App.css";
import { BrowserRouter, Routes, Route, useLocation } from "react-router-dom";

function AppContent() {
  const location = useLocation(); // Now inside BrowserRouter context

  return (
    <div className="A">
      {/* Conditionally render Nav based on the current path */}
      {location.pathname !== "/classrooms" && <Nav />}
      <main className="form-signin">
        <Routes>
          <Route path="/" exact element={<Home />} />
          <Route path="/login" element={<Login />} />
          <Route path="/signup" element={<SignUp />} />
          <Route path="/classrooms" element={<Classrooms />} />
        </Routes>
      </main>
    </div>
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