import { ResponsivePie } from '@nivo/pie';
import { tokens } from "../../theme";
import { useTheme } from "@mui/material";
import { useState, useEffect } from "react";

const PieChart = () => {
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
                                id: endpoint.key,
                                label: endpoint.key,
                                value: data.value || 0, // Ensure the API provides a numeric value
                            }));
                        })
                    )
                );

                setData(responses);
            } catch (error) {
                console.error("Error fetching data:", error);
            }
        };

        fetchElectricityData();
    }, []);

    return (
        <ResponsivePie
            data={data}
            margin={{ top: 40, right: 80, bottom: 80, left: 80 }}
            innerRadius={0.5}
            padAngle={0.7}
            cornerRadius={3}
            colors={{ scheme: "nivo" }}
            borderWidth={1}
            borderColor={{ from: "color", modifiers: [["darker", 0.2]] }}
            radialLabelsSkipAngle={10}
            radialLabelsTextColor="#333333"
            radialLabelsLinkColor={{ from: "color" }}
            sliceLabelsSkipAngle={10}
            sliceLabelsTextColor="#333333"
        />
    );
};

export default PieChart;