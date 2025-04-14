import { useState, useEffect } from "react";
import { ResponsiveBar } from "@nivo/bar";
import { tokens } from "../../theme";
import { useTheme } from "@mui/material";

const BarChart = () => {
    const theme = useTheme();
    const colors = tokens(theme.palette.mode);
    
    const [data, setData] = useState([]);
    useEffect(() => {
            const fetchElectricityData = async () => {
                try {
                  const endpoints = [
                    { key: "Humidity Sensor", url: "http://localhost:8081/electricity?id=2&type=Humidity" },
                    { key: "Temperature Sensor", url: "http://localhost:8081/electricity?id=1&type=temperature" },
                    { key: "Light Sensor", url: "http://localhost:8081/electricity?id=3&type=light" },
                    { key: "Fan", url: "http://localhost:8081/electricity?id=4&type=fan" },
                    { key: "Conditioner", url: "http://localhost:8081/electricity?id=5&type=conditioner" },
                    { key: "Projector", url: "http://localhost:8081/electricity?id=6&type=projector" },
                 ];
                  const response = await Promise.all(
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
                    return res.json().then((data) => ({
                        key: endpoint.key,
                        data // Assuming the response has a 'value' field
                    }));
                    }))
                    );
                    console.log("API Responses:", response);
                    const combinedData = response.reduce((acc, { key, data }) => {
                        data.forEach((item) => {
                          const existing = acc.find((entry) => entry.country === item.country);
                          if (existing) {
                            existing[key] = item.value;
                          } else {
                            acc.push({ country: item.country, [key]: item.value });
                          }
                        });
                        return acc;
                      }, []);
              
                      console.log("Combined Data:", combinedData);
                      setData(combinedData);
                    } catch (error) {
                      console.error("Error fetching data:", error);
                      setData([]);
                    }
                  };
                    
              fetchElectricityData();
            }, []);
    return (
        <ResponsiveBar
        data={data}
        keys={[
            'Humidity Sensor',
            'Temperature Sensor',
            'Light Sensor',
            'Fan',
            'Conditioner',
            'Projector',
        ]}
        indexBy="country"
        margin={{ top: 50, right: 130, bottom: 50, left: 60 }}
        padding={0.3}
        valueScale={{ type: 'linear' }}
        indexScale={{ type: 'band', round: true }}
        colors={{ scheme: 'nivo' }}
        defs={[
            {
                id: 'dots',
                type: 'patternDots',
                background: 'inherit',
                color: '#38bcb2',
                size: 4,
                padding: 1,
                stagger: true
            },
            {
                id: 'lines',
                type: 'patternLines',
                background: 'inherit',
                color: '#eed312',
                rotation: -45,
                lineWidth: 6,
                spacing: 10
            }
        ]}
        fill={[
            {
                match: {
                    id: 'fries'
                },
                id: 'dots'
            },
            {
                match: {
                    id: 'sandwich'
                },
                id: 'lines'
            }
        ]}
        borderColor={{
            from: 'color',
            modifiers: [
                [
                    'darker',
                    1.6
                ]
            ]
        }}
        axisTop={null}
        axisRight={null}
        axisBottom={{
            tickSize: 5,
            tickPadding: 5,
            tickRotation: 0,
            legend: 'country',
            legendPosition: 'middle',
            legendOffset: 32,
            truncateTickAt: 0
        }}
        axisLeft={{
            tickSize: 5,
            tickPadding: 5,
            tickRotation: 0,
            legend: 'food',
            legendPosition: 'middle',
            legendOffset: -40,
            truncateTickAt: 0
        }}
        labelSkipWidth={12}
        labelSkipHeight={12}
        labelTextColor={{
            from: 'color',
            modifiers: [
                [
                    'darker',
                    1.6
                ]
            ]
        }}
        legends={[
            {
                dataFrom: 'keys',
                anchor: 'bottom-right',
                direction: 'column',
                justify: false,
                translateX: 120,
                translateY: 0,
                itemsSpacing: 2,
                itemWidth: 100,
                itemHeight: 20,
                itemDirection: 'left-to-right',
                itemOpacity: 0.85,
                symbolSize: 20,
                effects: [
                    {
                        on: 'hover',
                        style: {
                            itemOpacity: 1
                        }
                    }
                ]
            }
        ]}
        role="application"
        ariaLabel="Nivo bar chart demo"
        barAriaLabel={e=>e.id+": "+e.formattedValue+" in country: "+e.indexValue}
    />
    );
    }
export default BarChart;
