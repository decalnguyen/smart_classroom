import React, { useState } from "react";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import Dashboard from "./components/Dashboard";
import Home from "./components/Home";
import Nav from "./components/Nav";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";

function App() {
 return (
    <div className="App">
        <Nav />
      <main class="form-signin">
        <BrowserRouter>
          <Routes>
            <Route path="/login" component={Login} />
            <Route path="/signup" component={SignUp} />
            <Route path="/home" exact component={Home} />
          </Routes>
        </BrowserRouter>
      </main>
    </div>
 );
}

export default App;