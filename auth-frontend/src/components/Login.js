import React, { useState } from 'react';
import { useAuth } from "../AuthContext";

const Login = () => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const { login, error } = useAuth();

  const submit = async (e) => {
    e.preventDefault();
    await login(username, password);
  };

  return (
    <form onSubmit={submit}>
      <h1 className="h3 mb-3 fw-normal">Please sign in</h1>
      <input
        type="text"
        value={username}
        onChange={(e) => setUsername(e.target.value)}
        className="form-control"
        placeholder="name@example.com"
        required
      />
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        className="form-control"
        placeholder="Password"
        required
      />
      <button className="btn btn-primary w-100 btn-lg" type="submit">
        Sign in
      </button>
      {error && <div style={{ color: "red" }}>{error}</div>}
    </form>
  );
};

export default Login;