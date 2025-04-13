import {Box, Typography, useTheme} from "@mui/material";
import Header from "../Header";
import {tokens} from "../../theme";
import {DataGrid, GridToolbar} from "@mui/x-data-grid";
import { useState, useEffect } from "react";


const Attandance = () => {
    const theme = useTheme();
    const colors = tokens(theme.palette.mode);

    const columns = [
        { field: "student_id", headerName: "ID" },
        { field: "student_name", headerName: "Name", flex: 1, cellClassName: "name-column--cell" },
        { field: "age", headerName: "Age", type: "number", headerAlign: "left", align: "left" },
        { field: "phone", headerName: "Phone Number", flex: 1 },
        { field: "email", headerName: "Email", flex: 1 },
      ];
    
    const [students, setStudents] = useState([]);

    useEffect(() => {
        const fetchStudentData = async () => {
            try {
              const response = await fetch("http://localhost:8081/students", {
                method: "GET",
                headers: {
                  "Content-Type": "application/json",
                },
                credentials: "include",
              });
      
              if (!response.ok) {
                throw new Error("Failed to fetch student data");
              }
      
              const data = await response.json();
              console.log("Fetched student Data:", data);
      
              // Ensure data is an array
              if (Array.isArray(data)) {
                setStudents(data);
                console.log("Updated Student State:", data); // Debugging log
              } 
            }catch (error) {
                console.error("Error fetching student data:", error);
                setStudents([]);
                console.log("Error fetching student data:", error); // Debugging log
            }
        };
          fetchStudentData();
        }, []);
    return (
        <Box m="20px">
            <Header title="Classroom Students" subtitle="Managing the classroom students" />
            <Box m= "40px 0 0 0" 
                height="75vh" 
                sx={{
                    "& .MuiDataGrid-root": {
                        border: "none",
                    },
                    "& .MuiDataGrid-cell": {
                        borderBottom: "none",
                    },
                    "& .name-column--cell": {
                        color: colors.greenAccent[300],
                    },
                    "& .MuiDataGrid-columnHeaders": {
                        backgroundColor: colors.blueAccent[700],
                        borderBottom: "none",
                    },
                    "& .MuiDataGrid-virtualScroller": {
                        backgroundColor: colors.primary[400],
                    },
                    "& .MuiDataGrid-footerContainer": {
                        borderTop: "none",
                        backgroundColor: colors.blueAccent[700],
                    },
                    "& .MuiDataGrid-toolbarContainer .MuiButton-text": {
                        color: `${colors.grey[100]} !important`,
                    },
                }}
               >
                <DataGrid rows={students} 
                columns={columns} 
                getRowId={(row) => row.student_id} 
                components={{ Toolbar: GridToolbar }}
                />
            </Box>
        </Box> 
    )
}
export default Attandance;