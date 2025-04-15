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
                    { key: "Light Sensor", url: "http://localhost:8081/electricity?id=3&type=Light" },
                    { key: "Fan", url: "http://localhost:8081/electricity?id=4&type=fan" },
                    { key: "Conditioner", url: "http://localhost:8081/electricity?id=5&type=conditioner" },
                    { key: "Projector", url: "http://localhost:8081/electricity?id=6&type=Projector" },
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
                            return res.json().then((data) => ({
                                key: endpoint.key,
                                data: Array.isArray(data) ? data : [], // Fallback to an empty array if data is not an array
                            }));
                        })
                    )
                );
    
                console.log("API Responses:", responses);
    
                const combinedData = responses.reduce((acc, { key, data }) => {
                    if (!Array.isArray(data)) {
                        console.error(`Data for key "${key}" is not an array:`, data);
                        return acc;
                    }
    
                    data.forEach((item) => {
                        const existing = acc.find((entry) => entry.device_type === item.device_type);
                        if (existing) {
                            existing[key] = item.value;
                        } else {
                            acc.push({
                                device_type: item.device_type,
                                [key]: item.value,
                                ...Object.fromEntries(
                                    endpoints.map((e) => [e.key, e.key === key ? item.value : 0])
                                ),
                            });
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
        <div style={{ height: "500px", width: "1000px" }}>
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
        indexBy="device_type"
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
            legend: 'Electricity Consumption',
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
        barAriaLabel={e=>e.id+": "+e.formattedValue+" in device type: "+e.indexValue}
    />
    </div>
    );
    }
export default BarChart;
