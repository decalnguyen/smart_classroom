import React from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "./AuthContext";
import Login from "./components/Login";
import SignUp from "./components/SignUp";
import Dashboard from "./components/dashboard/dashboard";
import Student from "./components/student/student";
import Attandance from "./components/attandance/attandance";
import Calendar from "./components/calendar/calendar";
import BarChart from "./components/charts/BarChart";
import PieChart from "./components/charts/PieChart";
import LineChart from "./components/charts/LineChart";
const PublicRoutes = [
    {
        path: "/login",
        element: <Login />,
    },
    {
        path: "/signup",
        element: <SignUp />,
    },
];
const PrivateRoutes = [
    {
        path: "/dashboard",
        element: <Dashboard />,
    },
    {
        path: "/students",
        element: <Student />,
    },
    {
        path: "/attandance",
        element: <Attandance />,
    },
    {
        path: "/calendar",
        element: <Calendar />,
    },
    {
        path: "/barchart",
        element: <BarChart />,
    },
    {
        path: "/piechart",
        element: <PieChart />,
    },
    {
        path: "/linechart",
        element: <LineChart />,
    },
];



export { PrivateRoutes, PublicRoutes };