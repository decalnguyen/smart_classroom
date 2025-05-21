import React, { useEffect, useState } from "react";
import "react-calendar/dist/Calendar.css";
import "../App.css"; // optional custom styling
const daysOfWeek = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

const CalendarComponent = () => {
  const [weekly, setWeekly] = useState({
    Monday: [],
    Tuesday: [],
    Wednesday: [],
    Thursday: [],
    Friday: [],
    Saturday: [],
    Sunday: []
  });
  const role = "student";

  useEffect(() => {
    // Replace this URL with your real API endpoint
   const token = localStorage.getItem("token"); // hoặc nơi bạn lưu token
  fetch("http://localhost:8081/schedules", {
    headers: {
      Authorization: token ? token : "", // hoặc `Bearer ${token}` nếu backend yêu cầu
    },
    credentials: "include", // nếu backend dùng cookie
  })
    .then((res) => res.json())
    .then((data) => {
      setWeekly(data);
    })
    .catch(() => {
      // fallback or show error
    });
  }, []);

 return (
  <div className="scroll-x">
    <div className="flex justify-between items-center mb-4 min-w-[1000px]">
      <h2 className="text-2xl font-bold min-w-[1000px]">{role.charAt(0).toUpperCase() + role.slice(1)}'s Schedule</h2>
    </div>
    <div className="weekly-grid">
      {daysOfWeek.map((day) => {
        const sessions = weekly[day] || [];
        return (
          <div key={day} className="bg-white shadow rounded-xl p-4">
            <h2 className="text-lg font-semibold mb-2">{day}</h2>
            {sessions.length === 0 ? (
              <p className="text-sm text-gray-500">No classes</p>
            ) : (
              Array.isArray(sessions) && sessions.map((session, idx) => (
                <div key={idx} className="mb-3 p-3 border rounded-lg bg-blue-50">
                  <p className="font-semibold text-blue-800">{session.time}</p>
                  <p className="text-sm text-gray-700">{session.title}</p>
                  <p className="text-sm text-gray-600">{session.desc}</p>
                  {session.room && session.room !== "" && (
                    <p className="text-xs text-gray-500 italic">Room: {session.room}</p>
                  )}
                </div>
              ))
            )}
          </div>
        );
      })}
    </div>
  </div>
);
};
export default CalendarComponent;