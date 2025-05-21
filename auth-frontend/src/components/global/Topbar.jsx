import {Box, IconButton, useTheme} from "@mui/material";
import {useContext} from "react";
import {ColorModeContext, tokens} from "../../theme";
import InputBase from "@mui/material/InputBase";
import LightModeOutlinedIcon from "@mui/icons-material/LightModeOutlined";
import DarkModeOutlinedIcon from "@mui/icons-material/DarkModeOutlined";
import NotificationsNoneOutlinedIcon from "@mui/icons-material/NotificationsNoneOutlined";
import SettingsOutlinedIcon from "@mui/icons-material/SettingsOutlined";
import SearchIcon from "@mui/icons-material/Search";
import PersonOutlinedIcon from "@mui/icons-material/PersonOutlined";
import LogoutIcon from '@mui/icons-material/Logout';
import {useAuth} from "../../AuthContext"; // Adjust the import path as necessary
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import React, { useState } from "react";
import {jwtDecode} from "jwt-decode";

const Topbar = () => {
    const theme = useTheme();
    const colors = tokens(theme.palette.mode);
    const colorMode = useContext(ColorModeContext);
    const { logout } = useAuth();

    // Notification state
    const [anchorEl, setAnchorEl] = useState(null);
    const [notifications, setNotifications] = useState([]);
    const [loading, setLoading] = useState(false);

    const token = localStorage.getItem("auth_token");
    let accountId = "";
    if (token) {
        try {
            const decoded = jwtDecode(token);
            accountId = decoded.account_id || decoded.AccountID || "";
        } catch (e) {
            accountId = "";
        }
    }

    const handleNotificationClick = async (event) => {
        setAnchorEl(event.currentTarget);
        setLoading(true);
        try {
            const res = await fetch(`http://localhost:8081/notifications?account_id=${accountId}`, {
                credentials: "include",
            });
            if (!res.ok) throw new Error("Failed to fetch notifications");
            const data = await res.json();
            console.log("Notifications data:", data);
            setNotifications(data);
        } catch (err) {
            setNotifications([]);
        }
        setLoading(false);
    };


    const handleNotificationClose = () => {
        setAnchorEl(null);
    };

    return (
        <Box display="flex" justifyContent="space-between" alignItems="center" p={2}>
            {/* Search Bar */}
            <Box display="flex" backgroundColor={colors.primary[400]} borderRadius="3px">
                <InputBase sx={{ ml: 2, flex: 1 }} placeholder="Search" />
                <IconButton type="button" sx={{ p: 1 }}>
                    <SearchIcon />
                </IconButton>
            </Box>

            {/* Icons */}
            <Box display="flex" ml="2000px">
                <IconButton onClick={colorMode.toggleColorMode}>
                    {theme.palette.mode === "dark" ? (
                        <DarkModeOutlinedIcon />
                    ) : (
                        <LightModeOutlinedIcon />
                    )}
                </IconButton>
                <IconButton onClick={handleNotificationClick}>
                    <NotificationsNoneOutlinedIcon />
                </IconButton>
                <Menu
                    anchorEl={anchorEl}
                    open={Boolean(anchorEl)}
                    onClose={handleNotificationClose}
                >
                    {loading ? (
                        <MenuItem disabled>Loading...</MenuItem>
                    ) : notifications.length === 0 ? (
                        <MenuItem disabled>No notifications</MenuItem>
                    ) : (
                        notifications.map((note, idx) => (
                            <MenuItem key={idx}>
                                {note.title ? <b>{note.title}: </b> : null}
                                {note.message || note}
                            </MenuItem>
                        ))
                    )}
                </Menu>
                <IconButton>
                    <SettingsOutlinedIcon />
                </IconButton>
                <IconButton>
                    <PersonOutlinedIcon />
                </IconButton>
                <IconButton>
                    <LogoutIcon onClick={logout} />
                </IconButton>
            </Box>
        </Box>
    );
};

export default Topbar;