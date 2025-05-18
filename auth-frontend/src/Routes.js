// Routes.js
import React from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "./AuthContext.jsx";

export const PrivateRoutes = ({ element }) => {
  const { token } = useAuth();
  return token ? element : <Navigate to="/login" />;
};

export const PublicRoutes = ({ element }) => {
  const { token } = useAuth();
  return !token ? element : <Navigate to="/dashboard" />;
};