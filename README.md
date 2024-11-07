# FealtyX Backend Assignment

This project is a simple REST API built with Go that performs CRUD operations on a list of students. It also integrates with Ollama to generate AI-based summaries for student profiles.

## Features

- Create, Read, Update, and Delete student records
- Generate AI-based summaries of student profiles using Ollama
- In-memory data storage
- Concurrent request handling
- Input validation

## Prerequisites

- Go 1.17 or later
- Ollama installed locally (for AI-based summaries)

## Setup

1. Clone the repository:
   ```
   git clone https://github.com/yuvraj1016/fealtyX-backend-assignment.git
   cd fealtyX-backend-assignment
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Run the application:
   ```
   go run main.go
   ```

The server will start on `http://localhost:8080` by default. You can change the port by setting the `PORT` environment variable.

## API Endpoints

### Create a new student
- **POST** `/students`
- Body: `{"id": 12345678, "name": "John Doe", "age": 20, "email": "john@example.com"}`

### Get all students
- **GET** `/students`

### Get a student by ID
- **GET** `/students/{id}`

### Update a student by ID
- **PUT** `/students/{id}`
- Body: `{"id": 12345678, "name": "John Doe", "age": 21, "email": "john.updated@example.com"}`

### Delete a student by ID
- **DELETE** `/students/{id}`

### Generate a summary of a student by ID
- **GET** `/students/{id}/summary`

## Data Model

Student:
- ID (8-digit integer)
- Name (string)
- Age (integer)
- Email (string)

## Error Handling

The API returns appropriate HTTP status codes and error messages for various scenarios, such as:
- 400 Bad Request: For invalid input data
- 404 Not Found: When a student with the specified ID does not exist
- 409 Conflict: When trying to create a student with an ID that already exists

## Ollama Integration

The application integrates with Ollama to generate summaries of student profiles. Make sure Ollama is running locally on the default port (11434) or set the `OLLAMA_PORT` environment variable if it's running on a different port.
