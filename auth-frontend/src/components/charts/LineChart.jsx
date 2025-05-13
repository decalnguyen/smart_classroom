import React, { useState, useEffect } from "react";
import { ResponsiveLine } from "@nivo/line";
import { tokens } from "../../theme";
import { useTheme } from "@mui/material";

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

        // Transform API responses into the required format for ResponsiveLine
        const transformedData = responses.map((response, index) => ({
          id: endpoints[index].key,
          data: response.map((item) => ({
            x: item.timestamp || `Point ${item.id}`, // Use a timestamp or fallback to a generic label
            y: item.value || 0, // Use the value or fallback to 0
          })),
        }));

        setData(transformedData);
      } catch (error) {
        console.error("Error fetching data:", error);
        setData([]); // Fallback to an empty array on error
      }
    };

    fetchSensorData();
  }, []);

  return (
    <ResponsiveLine
      data={data}
      theme={{
        axis: {
          domain: {
            line: {
              stroke: colors.grey[100],
            },
          },
          ticks: {
            line: {
              stroke: colors.grey[100],
              strokeWidth: 1,
            },
            text: {
              fill: colors.grey[100],
            },
          },
        },
        legends: {
          text: {
            fill: colors.grey[100],
          },
        },
      }}
      margin={{ top: 50, right: 110, bottom: 50, left: 60 }}
      xScale={{ type: "point" }}
      yScale={{
        type: "linear",
        min: "auto",
        max: "auto",
        stacked: true,
        reverse: false,
      }}
      axisTop={null}
      axisRight={null}
      axisBottom={{
        tickSize: 5,
        tickPadding: 5,
        tickRotation: 0,
        legend: "Time",
        legendOffset: 36,
        legendPosition: "middle",
      }}
      axisLeft={{
        tickSize: 5,
        tickPadding: 5,
        tickRotation: 0,
        legend: "Value",
        legendOffset: -40,
        legendPosition: "middle",
      }}
      pointSize={10}
      pointColor={{ theme: "background" }}
      pointBorderWidth={2}
      pointBorderColor={{ from: "serieColor" }}
      pointLabelYOffset={-12}
      useMesh={true}
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
    />
  );
};

export default LineChart;