package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Student struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

var (
	students   = make(map[int]Student)
	studentsMu sync.RWMutex
	ollamaHost string
	useOllama  bool
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/students", createStudent).Methods("POST")
	r.HandleFunc("/students", getAllStudents).Methods("GET")
	r.HandleFunc("/students/{id}", getStudent).Methods("GET")
	r.HandleFunc("/students/{id}", updateStudent).Methods("PUT")
	r.HandleFunc("/students/{id}", deleteStudent).Methods("DELETE")
	r.HandleFunc("/students/{id}/summary", getStudentSummary).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ollamaHost = os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	// Check if Ollama is available
	useOllama = checkOllamaAvailability()
	if useOllama {
		log.Println("Ollama is available and will be used for generating summaries.")
	} else {
		log.Println("Ollama is not available. Fallback mechanism will be used for generating summaries.")
	}

	log.Printf("Server starting on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func checkOllamaAvailability() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ollamaHost+"/api/tags", nil)
	if err != nil {
		log.Printf("Error creating request to check Ollama availability: %v", err)
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error checking Ollama availability: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}


func validateID(id int) bool {
	return id >= 10000000 && id <= 99999999
}

func createStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	err := json.NewDecoder(r.Body).Decode(&student)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !validateID(student.ID) {
		http.Error(w, "Invalid student ID: must be an 8-digit integer", http.StatusBadRequest)
		return
	}

	if student.Name == "" || student.Age <= 0 || student.Email == "" {
		http.Error(w, "Invalid student data", http.StatusBadRequest)
		return
	}

	studentsMu.Lock()
	if _, exists := students[student.ID]; exists {
		studentsMu.Unlock()
		http.Error(w, "Student ID already exists", http.StatusConflict)
		return
	}
	students[student.ID] = student
	studentsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(student)
}

func getAllStudents(w http.ResponseWriter, r *http.Request) {
	studentsMu.RLock()
	studentList := make([]Student, 0, len(students))
	for _, student := range students {
		studentList = append(studentList, student)
	}
	studentsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(studentList)
}

func getStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil || !validateID(id) {
		http.Error(w, "Invalid student ID: must be an 8-digit integer", http.StatusBadRequest)
		return
	}

	studentsMu.RLock()
	student, ok := students[id]
	studentsMu.RUnlock()

	if !ok {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

func updateStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil || !validateID(id) {
		http.Error(w, "Invalid student ID: must be an 8-digit integer", http.StatusBadRequest)
		return
	}

	var student Student
	err = json.NewDecoder(r.Body).Decode(&student)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if student.ID != id {
		http.Error(w, "Student ID in body does not match URL", http.StatusBadRequest)
		return
	}

	if student.Name == "" || student.Age <= 0 || student.Email == "" {
		http.Error(w, "Invalid student data", http.StatusBadRequest)
		return
	}

	studentsMu.Lock()
	if _, ok := students[id]; !ok {
		studentsMu.Unlock()
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	students[id] = student
	studentsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

func deleteStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil || !validateID(id) {
		http.Error(w, "Invalid student ID: must be an 8-digit integer", http.StatusBadRequest)
		return
	}

	studentsMu.Lock()
	if _, ok := students[id]; !ok {
		studentsMu.Unlock()
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	delete(students, id)
	studentsMu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}
func getStudentSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil || !validateID(id) {
		http.Error(w, "Invalid student ID: must be an 8-digit integer", http.StatusBadRequest)
		return
	}

	studentsMu.RLock()
	student, ok := students[id]
	studentsMu.RUnlock()

	if !ok {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	log.Printf("Generating summary for student ID: %d", id)
	var summary string
	if useOllama {
		summary, err = generateSummaryWithOllama(student)
	} else {
		summary, err = generateFallbackSummary(student)
	}

	if err != nil {
		log.Printf("Error generating summary: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate summary: %v", err), http.StatusInternalServerError)
		return
	}

	if summary == "" {
		log.Printf("Generated summary is empty for student ID: %d", id)
		http.Error(w, "Generated summary is empty", http.StatusInternalServerError)
		return
	}

	log.Printf("Summary generated successfully for student ID: %d", id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"summary": summary})
}

func generateSummaryWithOllama(student Student) (string, error) {
	url := fmt.Sprintf("%s/api/generate", ollamaHost)
	prompt := fmt.Sprintf("Summarize this student profile using only the provided details. Be brief, accurate, and creative:\n\nProfile:\n- Name: %s\n- Age: %d\n- Email: %s\n\nNote: Make the summary catchy and to the point without adding any extra information.", student.Name, student.Age, student.Email)
	
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":  "llama2",
		"prompt": prompt,
		"stream": false,
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	log.Printf("Sending request to Ollama API: %s", url)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making POST request to Ollama: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Received response from Ollama API. Status: %s", resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API returned non-200 status code: %d, body: %s", resp.StatusCode, string(body))
	}

	log.Printf("Response body: %s", string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error decoding Ollama response: %v", err)
	}

	summary, ok := result["response"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format from Ollama: %+v", result)
	}

	if summary == "" {
		return "", fmt.Errorf("Ollama returned an empty summary")
	}

	log.Printf("Generated summary: %s", summary)
	return summary, nil
}

func generateFallbackSummary(student Student) (string, error) {
	summary := fmt.Sprintf("Student %s is %d years old and can be contacted at %s.", student.Name, student.Age, student.Email)
	return summary, nil
}