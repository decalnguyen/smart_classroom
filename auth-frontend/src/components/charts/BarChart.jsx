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
                  const response = await fetch("http://localhost:8081/electricity", {
                    method: "GET",
                    headers: {
                      "Content-Type": "application/json",
                    },
                    credentials: "include",
                  });
          
                  if (!response.ok) {
                    throw new Error("Failed to fetch electricity data");
                  }
          
                  const data = await response.json();
                  console.log("Fetched Electricity Data:", data);
          
                  // Ensure data is an array
                  if (Array.isArray(data)) {
                    setData(data);
                    console.log("Updated Electricity State:", data); // Debugging log
                  } 
                }catch (error) {
                    console.error("Error fetching Electricity data:", error);
                    setData([]);
                    console.log("Error fetching Electricity data:", error); // Debugging log
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
            'projector',
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
