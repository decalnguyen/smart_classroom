import React from "react";

function Dashboard({ onLogout }) {
  const handleLogout = () => {
    localStorage.removeItem("token"); // Remove token from localStorage
    onLogout(); // Notify parent component
  };

  return (
    <div>
      <h2>Dashboard</h2>
      <p>Welcome to the protected dashboard!</p>
      <button onClick={handleLogout}>Logout</button>
    </div>
  );
}

export default Dashboard;