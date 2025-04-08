import React from "react";
import { Link } from "react-router-dom";

function Nav({ currentPath }) {
  return (
    <nav>
      <ul>
        <li>
          <Link to="/">Home</Link>
        </li>
        {/* Hide Login and Signup links on the classrooms page */}
        {currentPath !== "/classrooms" && (
          <>
            <li>
              <Link to="/login">Login</Link>
            </li>
            <li>
              <Link to="/signup">Sign Up</Link>
            </li>
          </>
        )}
        <li>
          <Link to="/classrooms">Classrooms</Link>
        </li>
      </ul>
    </nav>
  );
}

export default Nav;