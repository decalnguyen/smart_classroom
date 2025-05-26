import React, { useState, useEffect } from "react";
import { ResponsiveLine } from "@nivo/line";
import { tokens } from "../../theme";
import { useTheme } from "@mui/material";
const sensorColors = [
  "#ff6384", // Red
  "#36a2eb", // Blue
  "#4bc0c0", // Teal
  "#ffcd56", // Yellow
  "#9966ff", // Purple
  "#2ecc40", // Green
];
const MAX_POINTS = 20; // Số điểm tối đa muốn hiển thị trên mỗi đường

const LineChart = () => {
  const theme = useTheme();
  const colors = tokens(theme.palette.mode);
  const [data, setData] = useState([]);

  useEffect(() => {
    const fetchSensorData = async () => {
      try {
        const endpoints = [
          { key: "Humidity Sensor", url: "http://localhost:8081/sensor/sensor1" },
          { key: "Temperature Sensor", url: "http://localhost:8081/sensor/sensor2" },
          { key: "Light Sensor", url: "http://localhost:8081/sensor/sensor3" },
        ];

        const responses = await Promise.all(
          endpoints.map((endpoint) =>
            fetch(endpoint.url, {
              method: "GET",
              headers: {
                "Content-Type": "application/json",
              },
              credentials: "include",
            }).then((res) => {
              if (!res.ok) {
                throw new Error(`Failed to fetch ${endpoint.key} data`);
              }
              return res.json();
            })
          )
        );

        const transformedData = responses.map((response, index) => ({
          id: endpoints[index].key,
          data: response
            .slice(-MAX_POINTS) // Lấy N điểm mới nhất
            .map((item) => ({
              x: item.timestamp
                ? new Date(item.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
                : `Point ${item.id}`,
              y: item.value || 0,
            })),
        }));

        setData(transformedData);
      } catch (error) {
        console.error("Error fetching data:", error);
        setData([]);
      }
    };

    fetchSensorData();
    const interval = setInterval(fetchSensorData, 5000);

    return () => clearInterval(interval);
  }, []);

  return (
    <ResponsiveLine
      data={data}
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
      margin={{ top: 50, right: 120, bottom: 50, left: 60 }}
      xScale={{ type: "point" }}
      yScale={{
        type: "linear",
        min: "auto",
        max: "auto",
        stacked: false,
        reverse: false,
      }}
      axisTop={null}
      axisRight={null}
      axisBottom={{
        tickSize: 5,
        tickPadding: 5,
        tickRotation: 30,
        legend: "Time",
        legendOffset: 36,
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
      pointLabelYOffset={-12}
      useMesh={true}
      enableArea={true}
      areaOpacity={0.1}
      legends={[
        {
          anchor: "bottom-right",
          direction: "column",
          justify: false,
          translateX: 100,
          translateY: 0,
          itemsSpacing: 0,
          itemDirection: "left-to-right",
          itemWidth: 80,
          itemHeight: 20,
          itemOpacity: 0.75,
          symbolSize: 12,
          symbolShape: "circle",
          symbolBorderColor: "rgba(0, 0, 0, .5)",
          effects: [
            {
              on: "hover",
              style: {
                itemBackground: "rgba(0, 0, 0, .03)",
                itemOpacity: 1,
              },
            },
          ],
        },
      ]}
      tooltip={({ point }) => (
        <div>
          <strong>{point.serieId}</strong>
          <br />
          Time: {point.data.xFormatted}
          <br />
          Value: {point.data.yFormatted}
        </div>
      )}
    />
  );
};

export default LineChart;