# Use a slim Python image
FROM python:3.11-slim

# Set the working directory
WORKDIR /app

# Copy the requirements file and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application code
COPY . .

# Expose the port the service runs on
EXPOSE 5001

# Run the command to start the service
CMD ["python3", "embedding_service.py"]
