import React, { useState } from "react";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import Home from "./components/Home";
import Nav from "./components/Nav";
import "./App.css";
import { BrowserRouter, Routes, Route} from "react-router-dom";

function App() {
 return (
          <div className="A">
            <BrowserRouter>
              <Nav/>
              <main class="form-signin">
                <Routes>
                  <Route path="/" exact element={<Home />}/>
                  <Route path="/login" element={<Login />}/>
                  <Route path="/signup" element={<SignUp />}/>
                </Routes>
              </main>
            </BrowserRouter>
          </div>
        );
}

export default App;