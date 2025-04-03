import React, { useState } from "react";

const SignUp = () => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [redirect, setRedirect] = useState(false);
  const submit = async (e) => {
    e.preventDefault();
          const response = await fetch("http://localhost:8081/signup", {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              username,
              password,
            }),
          });
          setRedirect(true);
    }

    return (
      <form onSubmit={submit}>

      <h1 class="h3 mb-3 fw-normal">Please sign up</h1>
        <input type="text" value={username} onChange={(e) => setUsername(e.target.value)} 
               class="form-control" placeholder="name@example.com" required/>

        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)}
               class="form-control" placeholder="Password" required/>

      <button class="btn btn-primary w-100 btn-lg" type="submit">Sign in</button>
      </form>
  );
}

export default SignUp;