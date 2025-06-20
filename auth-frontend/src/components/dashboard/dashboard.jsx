import { Box, Button, Icon, IconButton, Typography, useTheme } from "@mui/material";
import Header from "../../components/Header";
import { tokens } from "../../theme";
import LineChart from "../charts/LineChart";
import BarChart from "../charts/BarChart";
import PieChart from "../charts/PieChart";
import ProgressCircle from "../charts/ProgressCircle";
import StatBox from "../charts/StatBox";
import DownloadOutlinedIcon from "@mui/icons-material/DownloadOutlined";
import EmailIcon from "@mui/icons-material/Email";
import PointOfSaleIcon from "@mui/icons-material/PointOfSale";
import PersonAddIcon from "@mui/icons-material/PersonAdd";
import TrafficIcon from "@mui/icons-material/Traffic";
import { DataGrid } from "@mui/x-data-grid";
import { useEffect, useState } from "react";



const Dashboard = () => {
    const theme = useTheme();
    const colors = tokens(theme.palette.mode);
    
    const [studentsToday, setStudentsToday] = useState(0);
    const [attendanceCount, setAttendanceCount] = useState(0);
    const [peopleInRoom, setPeopleInRoom] = useState(0);
    const [eventWait, setEventWait] = useState(0);
    const [fanMode, setFanMode] = useState(0);
    const [ledMode, setLedMode] = useState(0);

    useEffect(() => {
        // Example API calls, replace URLs with your real endpoints
        // fetch("/api/students-today")
        //     .then(res => res.json())
        //     .then(data => setStudentsToday(data.count || 0));

        // fetch("/api/people-in-room")
        //     .then(res => res.json())
        //     .then(data => setPeopleInRoom(data.count || 0));

        // fetch("/api/event-wait")
        //     .then(res => res.json())
        //     .then(data => setEventWait(data.count || 0));
        // fetch("/api/fan/mode")
        //     .then(res => res.json())
        //     .then(data => setFanMode(data.mode || 0));

        // fetch("/api/led/mode")
        //     .then(res => res.json())
        //     .then(data => setLedMode(data.mode || 0));    
        function fetchAttendance() {
            fetch("http://localhost:8081/attendance")
                .then(res => res.json())
                .then(data => setAttendanceCount(Array.isArray(data) ? data.length : 0));
        }

        fetchAttendance(); // initial fetch

         const interval = setInterval(fetchAttendance, 5000); // fetch every 5 seconds

        return () => clearInterval(interval); // cleanup on unmount
    }, []);
    const handleFanModeChange = (mode) => {
        fetch("/api/fan/mode", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ mode }),
        })
        .then(res => res.json())
        .then(() => setFanMode(mode));
    };

    const handleLedModeChange = (mode) => {
        fetch("/api/led/mode", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ mode }),
        })
        .then(res => res.json())
        .then(() => setLedMode(mode));
    };
    return (
        <Box m="20px">
            <Box display="flex" justifyContent="space-between" alignItems="center">
                <Header title="DASHBOARD" subtitle="Welcome to your dashboard" />
            <Box>
                <Button
                    sx={{
                        backgroundColor: colors.blueAccent[700],
                        color: colors.grey[100],
                        padding: "10px 20px",
                        marginBottom: "20px",
                    }}
                >
                    <DownloadOutlinedIcon sx={{ mr: "10px" }} />
                    Download Report
                </Button>
            </Box>
            </Box>
                <Box display="grid" 
                gridTemplateColumns="repeat(12, 1fr)" 
                gridAutoRows="140px"
                gap="20px">
                    {/* ROW 1 */}
                    <Box gridColumn="span 3" 
                        backgroundColor={colors.primary[400]} 
                        display="flex"
                        alignItems="center"
                        justifyContent="center"
                    >
                        <StatBox
                            title={studentsToday}
                            subtitle="Total Students Today"
                            progress="0.75"
                            increase="+14%"
                            icon={
                                <PersonAddIcon sx={{ color: colors.greenAccent[600], fontSize: "26px" }} />
                            }
                        />
                    </Box>
                    <Box gridColumn="span 3" 
                        backgroundColor={colors.primary[400]} 
                        display="flex"
                        alignItems="center"
                        justifyContent="center"
                    >
                        <StatBox
                            title={attendanceCount}
                            subtitle="Attandanced Count"
                            progress="0.75"
                            increase="+14%"
                            icon={
                                <PersonAddIcon sx={{ color: colors.greenAccent[600], fontSize: "26px" }} />
                            }
                        />
                    </Box>
                    <Box gridColumn="span 3" 
                        backgroundColor={colors.primary[400]} 
                        display="flex"
                        alignItems="center"
                        justifyContent="center"
                    >
                      <StatBox
                        title={peopleInRoom}
                        subtitle="People in the room"
                        progress="0.75"
                        increase="+14%"
                        icon={
                            <PersonAddIcon sx={{ color: colors.greenAccent[600], fontSize: "26px" }} />
                        }
                    />
                    </Box>
                    <Box gridColumn="span 3" 
                        backgroundColor={colors.primary[400]} 
                        display="flex"
                        alignItems="center"
                        justifyContent="center"
                    >
                        <StatBox
                            title={eventWait}
                            subtitle="Event wait"
                            progress="0.75"
                            increase="+14%"
                            icon={
                                <EmailIcon sx={{ color: colors.greenAccent[600], fontSize: "26px" }} />
                            }
                        />
                    </Box>
                    {/*Row 2*/}
                    <Box 
                        gridColumn="span 8"
                        gridRow="span 2"
                        height={"470px"}
                        width={"100%"}
                        backgroundColor={colors.primary[400]}
                    >
                        <Box
                            
                            mt="25px"
                            p="0 30px"
                            display="flex"
                            justifyContent="space-between"
                            alignItems="center"
                        >
                            <Box display="flex" alignItems="center">
                                <Box mr={4}>
                                    <Typography variant="h5" fontWeight="600" color={colors.grey[100]}>
                                        Sensor Value
                                    </Typography>
                                    <Typography variant="h3" fontWeight="500" color={colors.greenAccent[500]}>
                                        1,234
                                    </Typography>
                                </Box>
                                <Box mx={2} backgroundColor={colors.primary[400]} display="flex" flexDirection="column" alignItems="center" justifyContent="center">
                                    <Typography variant="h6" color={colors.grey[100]}>Fan Mode</Typography>
                                    <Typography variant="h4" color={colors.greenAccent[500]}>{fanMode}</Typography>
                                    <Box mt={1}>
                                        {[0,1,2,3].map(mode => (
                                            <Button
                                                key={mode}
                                                variant={fanMode === mode ? "contained" : "outlined"}
                                                color="primary"
                                                onClick={() => handleFanModeChange(mode)}
                                                sx={{ m: 0.5 }}
                                            >
                                                {mode}
                                            </Button>
                                        ))}
                                    </Box>
                                </Box>
                                <Box mx={2} backgroundColor={colors.primary[400]} display="flex" flexDirection="column" alignItems="center" justifyContent="center">
                                    <Typography variant="h6" color={colors.grey[100]}>LED Mode</Typography>
                                    <Typography variant="h4" color={colors.greenAccent[500]}>{ledMode}</Typography>
                                    <Box mt={1}>
                                        {[0,1,2,3].map(mode => (
                                            <Button
                                                key={mode}
                                                variant={ledMode === mode ? "contained" : "outlined"}
                                                color="secondary"
                                                onClick={() => handleLedModeChange(mode)}
                                                sx={{ m: 0.5 }}
                                            >
                                                {mode}
                                            </Button>
                                        ))}
                                    </Box>
                                </Box>
                            </Box>
                            <Box>
                                <IconButton>
                                    <DownloadOutlinedIcon
                                    sx={{
                                        fontSize: "26px", color: colors.greenAccent[500]}} 
                                    />
                                </IconButton>
                            </Box>
                        </Box>
                        <Box height="300px" weight ="100%" ml="-20px">
                             <LineChart isDashboard={true}/> 
                        </Box>
                        <Box height="250px" weight ="500px"ml="-20px">
                             <BarChart isDashboard={true}/> 
                        </Box>
                    </Box>
            </Box>
        </Box>
    );
};
export default Dashboard;