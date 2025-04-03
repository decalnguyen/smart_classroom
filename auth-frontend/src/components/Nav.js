import  React from 'react';
import { Link } from 'react-router-dom';
const Nav = () => {
    return(
        <nav class="navbar navbar-expand-md navbar-dark bg-dark mb-4">
          <div class="container-fluid">
            <Link to="/" class="navbar-brand" >Home page</Link>
            <div>
              <ul class="navbar-nav me-auto mb-2 mb-md-0">
                <li class="nav-item">
                  <Link to="/login" class="nav-link" >Login</Link>
                </li>
                <li class="nav-item">
                  <Link to="/signup" class="nav-link" >Sign Up</Link>
                </li>
              </ul>
            </div>
          </div>
      </nav>
    );
}

export default Nav;