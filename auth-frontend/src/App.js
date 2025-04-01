import React, { useState } from "react";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import Dashboard from "./components/Dashboard";
import Nav from "./components/Nav";

function App() {
 return (
    <div className="App">
        <Nav />
      <main class="form-signin">
        <Login />
      </main>
    </div>
 );
}

export default App;