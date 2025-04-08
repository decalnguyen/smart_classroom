import React, { useState, useEffect } from "react";
import { Line } from "react-chartjs-2";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
} from "chart.js";

// Register required components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
);

const Classroom = () => {
  const [sensorData, setSensorData] = useState([]);
  const [sensorStatus, setSensorStatus] = useState("Loading...");
  const [students, setStudents] = useState([]);
  const [teacher, setTeacher] = useState(""); // State for teacher information

  useEffect(() => {
    // Fetch sensor data
    const fetchSensorData = async () => {
      try {
        const response = await fetch("http://localhost:8081/sensor/sensor3", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
        });

        if (!response.ok) {
          throw new Error("Failed to fetch sensor data");
        }

        const data = await response.json();
        console.log("Fetched Sensor Data:", data);

        // Ensure data is an array
        if (Array.isArray(data)) {
          setSensorData(data);
          const isSensorActive = data.some((sensor) => sensor.status === "active");
          setSensorStatus(isSensorActive ? "Active" : "Inactive");
        } else {
          console.error("API response is not an array");
          setSensorData([]);
          setSensorStatus("Error fetching data");
        }
      } catch (error) {
        console.error("Error fetching sensor data:", error);
        setSensorData([]);
        setSensorStatus("Error fetching data");
      }
    };

    // Fetch teacher data
    const fetchTeacherData = async () => {
      try {
        const response = await fetch("http://localhost:8081/teachers", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
        });
    
        if (!response.ok) {
          throw new Error("Failed to fetch teacher data");
        }
    
        const teacherData = await response.json();
        console.log("Fetched Teacher Data:", teacherData); // Debugging log
        setTeacher(teacherData);
        console.log("Updated Teacher State:", teacherData); // Debugging log
      } catch (error) {
        console.error("Error fetching teacher data:", error);
        setTeacher(null);
      }
    };

    // Fetch data initially
    fetchSensorData();
    fetchTeacherData();

    // Set up polling to fetch sensor data every 5 seconds
    const interval = setInterval(fetchSensorData, 5000);

    // Cleanup interval on component unmount
    return () => clearInterval(interval);
  }, []);

  // Filter sensor data to include only 5-minute intervals
  const filteredSensorData = sensorData.filter((sensor) => {
    const timestamp = new Date(sensor.timestamp);
    return timestamp.getMinutes() % 5 === 0; // Include only timestamps divisible by 5 minutes
  });

  const chartData = {
    labels: filteredSensorData.map((sensor) => new Date(sensor.timestamp).toLocaleTimeString()), // Format timestamps for the X-axis
    datasets: [
      {
        label: "Sensor Readings",
        data: filteredSensorData.map((sensor) => sensor.value), // Map sensor values for the Y-axis
        borderColor: "rgba(75, 192, 192, 1)",
        backgroundColor: "rgba(75, 192, 192, 0.2)",
        fill: true,
      },
    ],
  };

  const chartOptions = {
    responsive: true,
    plugins: {
      legend: {
        display: true,
        position: "top",
      },
    },
  };

  if (sensorData.length === 0) {
    return <h2>No sensor data available</h2>;
  }

  return (
    <div className="classroom-page">
      <h1>Classroom Dashboard</h1>

      {/* Teacher Information Section */}
      <div className="teacher-section">
        <h2>Teacher Information</h2>
        {teacher ? (
          <div>
            <p><strong>Name:</strong> {teacher[0].teacher_name}</p>
            <p><strong>Subject:</strong> {teacher[0].subject}</p>
          </div>
        ) : (
          <p>Loading teacher information...</p>
        )}
      </div>

      {/* Sensor Data Section */}
      <div className="sensor-section">
        <h2>Sensor Data</h2>
        <div className="sensor-status">
          <h3>Sensor Status: {sensorStatus}</h3>
        </div>
        <div className="sensor-chart">
          <Line data={chartData} options={chartOptions} />
        </div>
      </div>

    </div>
  );
};

export default Classroom;