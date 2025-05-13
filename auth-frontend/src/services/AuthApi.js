import React, { useState } from "react";
import { Navigate } from "react-router-dom";

export const Login = async (username, password) => {
          e.preventDefault();
          await fetch("http://localhost:8081/login", {
            method: "POST",
            headers: { 
              "Content-Type": "application/json",
            },
            credentials: "include",
            body: JSON.stringify({
              username,
              password,
            }),
          });

}
export const SignUp = async (username, password) => {
          e.preventDefault();
          await fetch("http://localhost:8081/signup", {
            method: "POST",
            headers: { 
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              username,
              password,
            }),
          });
    
}
