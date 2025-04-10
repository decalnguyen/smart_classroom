import React, { useState, useEffect } from "react";
import Calendar from "react-calendar";
import "react-calendar/dist/Calendar.css";
import "../App.css"; // optional custom styling

const scheduleData = {
  Monday: [],
  Tuesday: [
    { time: "7:30–9:45", title: "CE213.P23.2 (Bi-weekly)", desc: "Digital System Design with HDL", room: "B2.06 (PM)" },
    { time: "13:00–15:15", title: "CE213.P23", desc: "Digital System Design with HDL", room: "B5.12" }
  ],
  Wednesday: [
    { time: "7:30–9:45", title: "CE122.P23", desc: "Technical Circuit Analysis", room: "C214 (CLC)" },
    { time: "13:45–15:15", title: "CE122.P23.1 (Bi-weekly)", desc: "Technical Circuit Analysis", room: "B5.08" }
  ],
  Thursday: [],
  Friday: [
    { time: "13:00–15:15", title: "CE232.P21", desc: "Embedded System Design (Wireless)", room: "B5.10" },
    { time: "16:15–17:00", title: "CE232.P21.2 (Bi-weekly)", desc: "Embedded System Design (Wireless)", room: "B4.04 (PM)" },
    { time: "17:45–20:45", title: "CE201.P21", desc: "Capstone Project 1", room: "" }
  ],
  Saturday: [
    { time: "7:30–9:00", title: "PE231.P23", desc: "Physical Education", room: "Sân Bóng Rổ" },
    { time: "17:45–20:45", title: "CE206.P21", desc: "Capstone Project 2", room: "" }
  ],
  Sunday: []
};

const CalendarComponent = () => {
  const [date, setDate] = useState(new Date());
  const [schedule, setSchedule] = useState([]);
  const [role, setRole] = useState("student");
  const [view, setView] = useState("calendar"); // 'calendar' or 'weekly'

  // Fetch API-based schedule (optional, use your existing API)
  useEffect(() => {
    const fetchSchedule = async () => {
      try {
        const response = await fetch(`/schedule?role=${role}`);
        if (!response.ok) throw new Error("Failed to fetch schedule");
        const data = await response.json();
        setSchedule(data);
      } catch (error) {
        console.error("Error fetching schedule:", error);
      }
    };

    // fetchSchedule(); // Uncomment to fetch from API
  }, [role]);

  // Hardcoded weekly schedule for now
  const weekly = scheduleData;

  const handleDateChange = (selectedDate) => setDate(selectedDate);

  const eventsForSelectedDate = schedule.filter(
    (event) => new Date(event.date).toDateString() === date.toDateString()
  );

  return (
    <div className="scroll-x">
      <div className="flex justify-between items-center mb-4 min-w-[1000px]">
        <h2 className="text-2xl font-bold min-w-[1000px]">{role.charAt(0).toUpperCase() + role.slice(1)}'s Schedule</h2>
        <div className="p-4 min-w-[1000px]">
          <button
            className={`px-4 py-1 rounded-l bg-blue-500 text-white ${view === "calendar" ? "opacity-100" : "opacity-60"}`}
            onClick={() => setView("calendar")}
          >
            Calendar View
          </button>
          <button
            className={` min-w-[1000px] text-white ${view === "weekly" ? "opacity-100" : "opacity-60"}`}
            onClick={() => setView("weekly")}
          >
            Weekly View
          </button>
        </div>
      </div>

      {view === "calendar" ? (
        <>
          <Calendar onChange={handleDateChange} value={date} />
          <div className="mt-4">
            <h3 className="text-lg font-semibold mb-2">Events on {date.toDateString()}:</h3>
            {eventsForSelectedDate.length > 0 ? (
              <ul className="list-disc pl-6">
                {eventsForSelectedDate.map((event, idx) => (
                  <li key={idx}>
                    <strong>{event.title}</strong> - {event.time}
                  </li>
                ))}
              </ul>
            ) : (
              <p>No events scheduled for this day.</p>
            )}
          </div>
        </>
      ) : (
        <div className="weekly-grid">
          {Object.entries(weekly).map(([day, sessions]) => (
            <div key={day} className="bg-white shadow rounded-xl p-4">
              <h2 className="text-lg font-semibold mb-2">{day}</h2>
              {sessions.length === 0 ? (
                <p className="text-sm text-gray-500">No classes</p>
              ) : (
                sessions.map((session, idx) => (
                  <div key={idx} className="mb-3 p-3 border rounded-lg bg-blue-50">
                    <p className="font-semibold text-blue-800">{session.time}</p>
                    <p className="text-sm text-gray-700">{session.title}</p>
                    <p className="text-sm text-gray-600">{session.desc}</p>
                    {session.room && (
                      <p className="text-xs text-gray-500 italic">Room: {session.room}</p>
                    )}
                  </div>
                ))
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default CalendarComponent;
