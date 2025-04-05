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
  const [sensorStatus, setSensorStatus] = useState("");

  useEffect(() => {
    const fetchSensorData = async () => {
      try {
        const response = await fetch("http://localhost:8081/sensor/sensor1", {
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
        setSensorData(data);

        const isSensorActive = data.some((sensor) => sensor.status === "active");
        setSensorStatus(isSensorActive ? "Active" : "Inactive");
      } catch (error) {
        console.error("Error fetching sensor data:", error);
        setSensorStatus("Error fetching data");
      }
    };

    fetchSensorData();
  }, []);

  const chartData = {
    labels: sensorData.map((sensor) => sensor.timestamp),
    datasets: [
      {
        label: "Sensor Readings",
        data: sensorData.map((sensor) => sensor.value),
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

  return (
    <div className="classroom-page">
      <h1>Classroom Sensor Dashboard</h1>
      <div className="sensor-status">
        <h2>Sensor Status: {sensorStatus}</h2>
      </div>
      <div className="sensor-chart">
        <Line data={chartData} options={chartOptions} />
      </div>
    </div>
  );
};

export default Classroom;