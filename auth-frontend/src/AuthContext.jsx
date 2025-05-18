import React, { createContext, useState, useContext, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Login, SignUp } from "./services/AuthApi.js";
const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
  const [token, setToken] = useState(
    localStorage.getItem("token") || null
  );
  const navigate = useNavigate(); // <-- Add this line

  useEffect(() => {
    if (token) {
      localStorage.setItem("token", token);
    } else {
      localStorage.removeItem("token");
    }
  }, [token]);

  const handleSignUp = async (user_name, password) => {
    const data = await SignUp(user_name, password);
    return data;
  };
  const handleLogin = async (user_name, password) => {
  const data = await Login(user_name, password);
  console.log("Login API response:", data); // Debug: see what you get
  if (data?.token) { // <-- Fix here
    setToken(data.token);
    localStorage.setItem("token", data.token);
    navigate("/dashboard");
  }
  return data;
};

  const handleLogout = () => {
    setToken(null);
    localStorage.removeItem("token");
    navigate("/login");
  };

  return (
    <AuthContext.Provider
      value={{
        token,
        signUp: handleSignUp,
        login: handleLogin,
        logout: handleLogout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);