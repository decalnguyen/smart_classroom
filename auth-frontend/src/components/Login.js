import React from 'react';
const Login = () => {
  return (
      <form>

      <h1 class="h3 mb-3 fw-normal">Please sign in</h1>
        <input type="email" class="form-control" placeholder="name@example.com" required/>

        <input type="password" class="form-control" placeholder="Password" required/>

      <button class="btn btn-primary w-100 btn-lg" type="submit">Sign in</button>
      </form>
  );
}
export default Login;