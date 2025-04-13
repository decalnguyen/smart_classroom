import {useState } from "react";
import { Box, 
    List,
    ListItem,
    ListItemText, 
    Typography, 
    useTheme } from "@mui/material";
import { tokens } from "../../theme";
import Header from "../Header";
import FullCalendar from "@fullcalendar/react";
import {formatDate} from "@fullcalendar/core";
import dayGridPlugin from "@fullcalendar/daygrid";
import timeGridPlugin from "@fullcalendar/timegrid";
import interactionPlugin from "@fullcalendar/interaction"; // needed for dayClick
import ListPlugin from "@fullcalendar/list";

const Calendar = () => {
    const theme = useTheme();
    const colors = tokens(theme.palette.mode);
    const [currentEvents, setCurrentEvents] = useState([]);

    const handleDateClick = (selected) => {
        const title = prompt("Please enter a new title for your event");
        const calendarApi = selected.view.calendar;
        calendarApi.unselect(); // clear date selection
        if (title) {
            calendarApi.addEvent({
                id: `${selected.dateStr}`,
                title,
                start: selected.startStr,
                end: selected.endStr,
                allDay: selected.allDay,
            });
        }
    }
    const handleEventClick = (selected) => {
        if (window.confirm(`Are you sure you want to delete the event '${selected.event.title}'`)) {
            selected.event.remove();
        }
    }
    return (
        <Box m="20px">
            <Header title="CALENDAR" subtitle="Full Calendar Interactive Page" />
            <Box display="flex" justifyContent="space-between">
                <Box flex="1 1 20%" borderRadius="4px" padding="15px" bgcolor={colors.primary[400]}>
                    <Typography variant="h5">Events</Typography>
                    <List>
                        {currentEvents.map((event) => (
                            <ListItem key={event.id} sx={{backgroundColor: colors.greenAccent[500], margin: "10px 0", borderRadius: "2px"}}>
                                <ListItemText 
                                    primary={event.title} 
                                    secondary={
                                        <Typography>
                                            {formatDate(event.start, {
                                                year: 'numeric',
                                                month: 'long',
                                                day: 'numeric',
                                            })}
                                        </Typography>} 
                                />
                            </ListItem>
                        ))}
                    </List>
                </Box>
                <Box flex="1 1 100%" ml="15px">
                    <FullCalendar
                        height="75vh"
                        plugins={[dayGridPlugin, timeGridPlugin, interactionPlugin, ListPlugin]}
                        headerToolbar={{
                            left: "prev,next today",
                            center: "title",
                            right: "dayGridMonth,timeGridWeek,timeGridDay,listMonth",
                        }}
                        initialView="dayGridMonth"
                        editable={true}
                        selectable={true}
                        selectMirror={true}
                        dayMaxEvents={true}
                        weekends={false}
                        initialEvents={[ // alternatively, use the setEvents method
                            { id: "1234", title: "All-day event", date: "2023-10-14" },
                            { id: "4321", title: "Timed event", date: "2023-10-28" },
                        ]}
                        select={handleDateClick}
                        eventClick={handleEventClick}
                        eventsSet={(events) => setCurrentEvents(events)} 
                    />
                </Box>
            </Box>
        </Box>
    )
}

export default Calendar;