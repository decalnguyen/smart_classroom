import React, { useState, useEffect } from "react";
import { ResponsiveLine } from "@nivo/line";
import { tokens } from "../../theme";
import { useTheme, Box, Button, TextField } from "@mui/material";

const sensorColors = [
  "#ff6384", "#36a2eb", "#4bc0c0", "#ffcd56", "#9966ff", "#2ecc40",
];

const MAX_POINTS = 20;

const LineChart = () => {
  const theme = useTheme();
  const colors = tokens(theme.palette.mode);
  const [data, setData] = useState([]);
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");

  const knownSensors = ["Temperature Sensor", "Humidity Sensor", "Light Sensor"];

  useEffect(() => {
    const ws = new WebSocket("ws://localhost:8081/ws/sensor");

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        const sensorName = msg.device_type; // 👈 use device_type instead of sensor_name

        if (!knownSensors.includes(sensorName)) return;

        const timestamp = new Date(msg.timestamp);
        if (isNaN(timestamp.getTime())) return;

        const point = {
          x: timestamp.toISOString(),
          y: typeof msg.value === "number" ? msg.value : 0,
        };

        setData((prevData) => {
          const newData = [...prevData];
          const sensorIndex = newData.findIndex(s => s.id === sensorName);

          if (!point.x || typeof point.y !== "number") return prevData;

          if (sensorIndex !== -1) {
            const sensorData = [...newData[sensorIndex].data, point];
            if (sensorData.length > MAX_POINTS) sensorData.shift();
            newData[sensorIndex] = { ...newData[sensorIndex], data: sensorData };
          } else {
            newData.push({ id: sensorName, data: [point] });
          }

          return newData;
        });
      } catch (error) {
        console.error("WebSocket error:", error);
      }
    };

    return () => ws.close();
  }, []);

  const fetchHistoricalData = async () => {
    try {
      const endpoints = [
        { key: "Humidity Sensor", url: `http://localhost:8081/sensor/sensor1?start=${startTime}&end=${endTime}` },
        { key: "Temperature Sensor", url: `http://localhost:8081/sensor/sensor2?start=${startTime}&end=${endTime}` },
        { key: "Light Sensor", url: `http://localhost:8081/sensor/sensor3?start=${startTime}&end=${endTime}` },
      ];

      const responses = await Promise.all(
        endpoints.map(endpoint =>
          fetch(endpoint.url, {
            method: "GET",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
          }).then(res => res.ok ? res.json() : [])
        )
      );

      const transformedData = responses.map((response, index) => ({
        id: endpoints[index].key,
        data: (response || []).map(item => ({
          x: new Date(item.timestamp).toISOString(),
          y: typeof item.value === "number" ? item.value : 0,
        })),
      }));

      setData(transformedData);
    } catch (error) {
      console.error("Error fetching historical data:", error);
    }
  };

  return (
    <>
      <Box mb={2} display="flex" gap={2}>
        <TextField
          label="Start Time"
          type="datetime-local"
          value={startTime}
          onChange={(e) => setStartTime(e.target.value)}
          InputLabelProps={{ shrink: true }}
        />
        <TextField
          label="End Time"
          type="datetime-local"
          value={endTime}
          onChange={(e) => setEndTime(e.target.value)}
          InputLabelProps={{ shrink: true }}
        />
        <Button variant="contained" onClick={fetchHistoricalData}>
          Fetch Historical Data
        </Button>
      </Box>

      <ResponsiveLine
        data={data.length > 0 ? data : [{
          id: "No Data",
          data: Array.from({ length: MAX_POINTS }, (_, i) => ({
            x: new Date(Date.now() - (MAX_POINTS - i) * 1000).toISOString(),
            y: 0,
          })),
        }]}
        theme={{
          axis: {
            domain: { line: { stroke: colors.grey[100] } },
            ticks: {
              line: { stroke: colors.grey[100], strokeWidth: 1 },
              text: { fill: colors.grey[100] },
            },
          },
          legends: { text: { fill: colors.grey[100] } },
          tooltip: {
            container: {
              background: "#222",
              color: "#fff",
              fontSize: 14,
            },
          },
        }}
        colors={sensorColors}
        margin={{ top: 50, right: 120, bottom: 60, left: 60 }}
        xScale={{
          type: "time",
          format: "%Y-%m-%dT%H:%M:%S.%LZ",
          useUTC: false,
          precision: "second",
        }}
        xFormat="time:%H:%M:%S"
        yScale={{ type: "linear", min: "auto", max: "auto", stacked: false }}
        axisTop={null}
        axisRight={null}
        axisBottom={{
          format: "%H:%M:%S",
          tickValues: "every 1 minute",
          tickSize: 5,
          tickPadding: 5,
          tickRotation: 30,
          legend: "Time",
          legendOffset: 40,
          legendPosition: "middle",
        }}
        axisLeft={{
          tickSize: 5,
          tickPadding: 5,
          tickRotation: 0,
          legend: "Sensor Value",
          legendOffset: -40,
          legendPosition: "middle",
        }}
        pointSize={10}
        pointColor={{ theme: "background" }}
        pointBorderWidth={2}
        pointBorderColor={{ from: "serieColor" }}
        useMesh={true}
        enableArea={true}
        areaOpacity={0.1}
        legends={[{
          anchor: "bottom-right",
          direction: "column",
          translateX: 100,
          itemWidth: 80,
          itemHeight: 20,
          symbolSize: 12,
          symbolShape: "circle",
          effects: [{
            on: "hover",
            style: {
              itemBackground: "rgba(0, 0, 0, .03)",
              itemOpacity: 1,
            },
          }],
        }]}
        tooltip={({ point }) => (
          <div>
            <strong>{point.serieId}</strong><br />
            Time: {new Date(point.data.x).toLocaleTimeString()}<br />
            Value: {point.data.y}
          </div>
        )}
      />
    </>
  );
};

export default LineChart;
