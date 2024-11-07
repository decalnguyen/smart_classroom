from deepface import DeepFace
import time
import pandas as pd
# Parse the pairs.txt file to create list of image pairs and labels
# Updated function to skip pairs with missing files
def load_lfw_pairs(pairs_file, lfw_dir="lfw"):
    image_pairs = []
    with open(pairs_file, "r") as f:
        lines = f.readlines()[1:]  # Skip the header line
        for line in lines:
            parts = line.strip().split()
            if len(parts) == 3:
                # Same person pair
                name, img1_idx, img2_idx = parts
                img1_path = os.path.join(lfw_dir, name, f"{name}_{int(img1_idx):04d}.jpg")
                img2_path = os.path.join(lfw_dir, name, f"{name}_{int(img2_idx):04d}.jpg")
                # Only add the pair if both images exist
                if os.path.exists(img1_path) and os.path.exists(img2_path):
                    image_pairs.append((img1_path, img2_path, True))
            elif len(parts) == 4:
                # Different people pair
                name1, img1_idx, name2, img2_idx = parts
                img1_path = os.path.join(lfw_dir, name1, f"{name1}_{int(img1_idx):04d}.jpg")
                img2_path = os.path.join(lfw_dir, name2, f"{name2}_{int(img2_idx):04d}.jpg")
                # Only add the pair if both images exist
                if os.path.exists(img1_path) and os.path.exists(img2_path):
                    image_pairs.append((img1_path, img2_path, False))
    return image_pairs

# Reload pairs with updated function
image_pairs = load_lfw_pairs("pairs.txt")
print("Loaded pairs:", image_pairs[:5])  # Display a few sample pairs to confirm

# Function to evaluate accuracy and speed on the LFW pairs with enforce_detection set to False
def evaluate_lfw_pairs(model_name, metric, image_pairs):
    correct_matches = 0
    total_time = 0

    for img1, img2, is_same in image_pairs:
        try:
            # Measure time for each comparison
            start_time = time.time()
            result = DeepFace.verify(
                img1_path=img1,
                img2_path=img2,
                model_name=model_name,
                distance_metric=metric,
                enforce_detection=False  # Bypass face detection errors
            )
            end_time = time.time()

            # Check if the model's prediction matches the actual label
            if result["verified"] == is_same:
                correct_matches += 1

            # Track time taken
            total_time += (end_time - start_time)
        except ValueError as e:
            print(f"Error processing pair {img1}, {img2}: {e}")

    # Calculate accuracy and average time
    accuracy = correct_matches / len(image_pairs)
    avg_time = total_time / len(image_pairs)
    return accuracy, avg_time

pairs_file = "pairs.txt"
models = ["VGG-Face", "Facenet", "OpenFace", "DeepFace", "ArcFace"]
distance_metrics = ["cosine", "euclidean", "euclidean_l2"]

results = []
image_pairs = load_lfw_pairs(pairs_file)
for model in models:
    for metric in distance_metrics:
        print(f"Evaluating model {model} with metric {metric}...")
        accuracy, avg_time = evaluate_lfw_pairs(model, metric, image_pairs[:100])  # Adjust number of pairs for speed
        results.append({
            "Model": model,
            "Metric": metric,
            "Accuracy": accuracy,
            "Average Time (s)": avg_time
        })

# Display results in a DataFrame

df_results = pd.DataFrame(results)
df_results = df_results.sort_values(by=["Accuracy", "Average Time (s)"], ascending=[False, True])

print(df_results)

