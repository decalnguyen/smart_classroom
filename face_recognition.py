# importing the cv2 library
import cv2
import numpy as np
from imgbeddings import imgbeddings
from PIL import Image
import psycopg2
import time

# Initialize the camera
cap = cv2.VideoCapture(0)
# loading the Haar Cascade algorithm file into alg variable
alg = "haarcascade_frontalface_default.xml"

#passing the algorithm to OpenCV
haar_cascade = cv2.CascadeClassifier(alg)

#loading the image path into file_name variable
# file_name = 'nhattoan.jpg'

# # reading the image
# img = cv2.imread(file_name, 0)

# # creating a black and white version of the image
# gray_img = cv2.cvtColor(img, cv2.COLOR_RGB2BGR)

# # detecting the faces
# faces = haar_cascade.detectMultiScale(gray_img, scaleFactor=1.05, minNeighbors=2, minSize=(100, 100))

# # for each face detected
# for x, y, w, h in faces:
#     # crop the image to select only the face
#     cropped_image = img[y : y + h, x : x + w]
    
#     # loading the target image path into target_file_name variable
#     target_file_name = 'nhattoan_detect.jpg'
#     cv2.imwrite(target_file_name, cropped_image)

# # loading the face image path into file_name variable
# file_name = 'nhattoan_detect.jpg'

# # opening the image
# img = Image.open(file_name)

# # loading the `imgbeddings`
# ibed = imgbeddings()

# # calculating the embeddings
# embedding = ibed.to_embeddings(img)[0]

# conn = psycopg2.connect("postgresql://nhattoan:test123@localhost:5432/cv_module")
# cur = conn.cursor()
# cur.execute('INSERT INTO pictures values (%s,%s)', (file_name, embedding.tolist()))
# conn.commit()
# conn.close()


while True:
    # Capture frame-by-frame
    ret, frame = cap.read()
    if not ret:
        print("Error reading frame from camera")
        break
    # Display the resulting frame
    cv2.imshow('frame', frame)

    # Wait for a key press
    key = cv2.waitKey(1) & 0xFF

    if key == ord('r'):
        # Capture the image when 'r' is pressed
        cv2.imwrite('captured_image.jpg', frame)
        print('Image captured')
        # loading the image path into file_name variable
        file_name = 'captured_image.jpg'

        # reading the image
        img = cv2.imread(file_name, 0)

        # creating a black and white version of the image
        gray_img = cv2.cvtColor(img, cv2.COLOR_RGB2BGR)

        # detecting the faces
        faces = haar_cascade.detectMultiScale(gray_img, scaleFactor=1.05, minNeighbors=2, minSize=(100, 100))
        # for each face detected in the image
        for x, y, w, h in faces:
            # crop the image to select only the face
            cropped_image = img[y : y + h, x : x + w]
            
            # Convert the NumPy array to a PIL image
            pil_image = Image.fromarray(cropped_image)
            
            ibed = imgbeddings()
            
            # calculating the embeddings5000
            slack_img_embedding = ibed.to_embeddings(pil_image)[0]

        conn = psycopg2.connect("postgresql://nhattoan:test123@localhost:5432/cv_module")
        cur = conn.cursor()
        string_rep = "[" + ",".join(str(x) for x in slack_img_embedding.tolist()) + "]"
        cur.execute("""SELECT picture, embedding <-> %s AS distance FROM pictures ORDER BY distance LIMIT 1;""", (string_rep,))

        # Fetch the result
        row = cur.fetchone()

        # Extract the picture and distance from the result
        picture = row[0]
        distance = row[1]

        print(1)

        if distance <= 8:
            print(f"Match found: {picture}")
            msg = bytes("OPEN", 'utf-8')
        else:
            print("Not detect")
    

    # Break the loop on 'q' key press
    if key == ord('q'):
        break

# When everything done, release the capture
cap.release()
cv2.destroyAllWindows()