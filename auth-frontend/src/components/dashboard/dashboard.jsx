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



const Dashboard = () => {
    const theme = useTheme();
    const colors = tokens(theme.palette.mode);

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
                            title="12,361"
                            subtitle="Total Students Today"
                            progress="0.75"
                            increase="+14%"
                            icon={
                            <PersonAddIcon sx={{ color: colors.greenAccent[600], fontSize: "26px" }} 
                            />
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
                            title="431,225"
                            subtitle="Attandanced Count"
                            progress="0.75"
                            increase="+14%"
                            icon={
                            <PersonAddIcon sx={{ color: colors.greenAccent[600], fontSize: "26px" }} 
                            />
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
                            title="32,441"
                            subtitle="People in the room"
                            progress="0.75"
                            increase="+14%"
                            icon={
                            <PersonAddIcon sx={{ color: colors.greenAccent[600], fontSize: "26px" }} 
                            />
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
                            title="32,441"
                            subtitle="Event wait"
                            progress="0.75"
                            increase="+14%"
                            icon={
                            <EmailIcon sx={{ color: colors.greenAccent[600], fontSize: "26px" }} 
                            />
                         }
                        />
                    </Box>
                    {/*Row 2*/}
                    <Box 
                        gridColumn="span 8"
                        gridRow="span 2"
                        backgroundColor={colors.primary[400]}
                    >
                        <Box
                            mt="25px"
                            p="0 30px"
                            display="flex"
                            justifyContent="space-between"
                            alignItems="center"
                        >
                            <Box>
                                <Typography variant="h5" fontWeight="600" color={colors.grey[100]}>
                                    Sensor Value
                                </Typography>
                                <Typography variant="h3" fontWeight="500" color={colors.greenAccent[500]}>
                                    1,234
                                </Typography>
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
                        <Box height="250px" weight ="500px"ml="-20px">
                             <LineChart isDashboard={true}/> 
                        </Box>
                    </Box>
            </Box>
        </Box>
    );
};
export default Dashboard;