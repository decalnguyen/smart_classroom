import cv2
import mediapipe as mp

from kuksa_client.grpc import Datapoint
from kuksa_client.grpc import DataEntry
from kuksa_client.grpc import DataType
from kuksa_client.grpc import EntryUpdate
from kuksa_client.grpc import Field
from kuksa_client.grpc import Metadata
from kuksa_client.grpc import VSSClient

# Initialize MediaPipe Hands
mp_hands = mp.solutions.hands
hands = mp_hands.Hands()
mp_draw = mp.solutions.drawing_utils

# Initialize webcam
cap = cv2.VideoCapture(0)

def setTrue():
    with VSSClient('127.0.0.1', 55555) as client:
        updates = (EntryUpdate(DataEntry(
            'Vehicle.IsPotholeDetected',
            value=Datapoint(value=True),
            metadata=Metadata(data_type=DataType.BOOLEAN),
        ), (Field.VALUE,)),)
        client.set(updates=updates)

def setFalse():
    with VSSClient('127.0.0.1', 55555) as client:
        updates = (EntryUpdate(DataEntry(
            'Vehicle.IsPotholeDetected',
            value=Datapoint(value=False),
            metadata=Metadata(data_type=DataType.BOOLEAN),
        ), (Field.VALUE,)),)
        client.set(updates=updates)


while cap.isOpened():
    ret, frame = cap.read()
    if not ret:
        break

    # Convert the frame to RGB
    rgb_frame = cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)

    # Process the frame and detect hands
    results = hands.process(rgb_frame)
    flag = 0
    # Draw hand landmarks
    if results.multi_hand_landmarks:
        for hand_landmarks in results.multi_hand_landmarks:
            mp_draw.draw_landmarks(frame, hand_landmarks, mp_hands.HAND_CONNECTIONS)

            print("hand detected")
            flag = 1
    print("1")            
    if flag == 1:
        setTrue()
    else:
        setFalse()


    # Display the frame
    # cv2.imshow('Hand Detection', frame)

    # Break loop on 'q' key press
    if cv2.waitKey(1) & 0xFF == ord('q'):
        break

# Release the webcam and close windows
cap.release()
cv2.destroyAllWindows()