// main.go
package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "sync"
    "github.com/gorilla/mux"
    _ "github.com/mattn/go-sqlite3" // Import the SQLite driver
)

// Student represents a student entity
type Student struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Age   int    `json:"age"`
    Email string `json:"email"`
}

// StudentStore manages student data with thread-safe operations
type StudentStore struct {
    sync.RWMutex
    students map[int]Student
    nextID   int
    db       *sql.DB
}

// NewStudentStore initializes a new StudentStore
func NewStudentStore(db *sql.DB) *StudentStore {
    return &StudentStore{
        students: make(map[int]Student),
        nextID:   1,
        db:       db,
    }
}

// ValidationError represents an input validation error
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

// Validate checks if student data is valid
func (s Student) Validate() []ValidationError {
    var errors []ValidationError

    if s.Name == "" {
        errors = append(errors, ValidationError{
            Field:   "name",
            Message: "Name is required",
        })
    }

    if s.Age < 0 || s.Age > 150 {
        errors = append(errors, ValidationError{
            Field:   "age",
            Message: "Age must be between 0 and 150",
        })
    }

    if s.Email == "" {
        errors = append(errors, ValidationError{
            Field:   "email",
            Message: "Email is required",
        })
    }

    return errors
}

type App struct {
    store *StudentStore
}

func (app *App) CreateStudent(w http.ResponseWriter, r *http.Request) {
    var student Student
    if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if errors := student.Validate(); len(errors) > 0 {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(errors)
        return
    }

    app.store.Lock()
    student.ID = app.store.nextID
    app.store.nextID++
    app.store.students[student.ID] = student
    app.store.Unlock()

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(student)
}

func (app *App) GetAllStudents(w http.ResponseWriter, r *http.Request) {
    app.store.RLock()
    students := make([]Student, 0, len(app.store.students))
    for _, student := range app.store.students {
        students = append(students, student)
    }
    app.store.RUnlock()

    json.NewEncoder(w).Encode(students)
}

func (app *App) GetStudent(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    app.store.RLock()
    student, exists := app.store.students[id]
    app.store.RUnlock()

    if !exists {
        http.Error(w, "Student not found", http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(student)
}

func (app *App) UpdateStudent(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    var student Student
    if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if errors := student.Validate(); len(errors) > 0 {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(errors)
        return
    }

    app.store.Lock()
    if _, exists := app.store.students[id]; !exists {
        app.store.Unlock()
        http.Error(w, "Student not found", http.StatusNotFound)
        return
    }

    student.ID = id
    app.store.students[id] = student
    app.store.Unlock()

    json.NewEncoder(w).Encode(student)
}

func (app *App) DeleteStudent(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    app.store.Lock()
    if _, exists := app.store.students[id]; !exists {
        app.store.Unlock()
        http.Error(w, "Student not found", http.StatusNotFound)
        return
    }

    delete(app.store.students, id)
    app.store.Unlock()

    w.WriteHeader(http.StatusNoContent)
}

func (app *App) GetStudentSummary(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    app.store.RLock()
    student, exists := app.store.students[id]
    app.store.RUnlock()

    if !exists {
        http.Error(w, "Student not found", http.StatusNotFound)
        return
    }

    summary := fmt.Sprintf("Student %s is %d years old with email %s.", student.Name, student.Age, student.Email)
    json.NewEncoder(w).Encode(map[string]string{"summary": summary})
}

func main() {
    db, err := sql.Open("sqlite3", "./students.db")
    if (err != nil) {
        log.Fatal(err)
    }
    defer db.Close()

    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS students (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT,
        age INTEGER,
        email TEXT
    )`)
    if err != nil {
        log.Fatal(err)
    }

    app := &App{
        store: NewStudentStore(db),
    }

    router := mux.NewRouter()

    router.HandleFunc("/students", app.CreateStudent).Methods("POST")
    router.HandleFunc("/students", app.GetAllStudents).Methods("GET")
    router.HandleFunc("/students/{id}", app.GetStudent).Methods("GET")
    router.HandleFunc("/students/{id}", app.UpdateStudent).Methods("PUT")
    router.HandleFunc("/students/{id}", app.DeleteStudent).Methods("DELETE")
    router.HandleFunc("/students/{id}/summary", app.GetStudentSummary).Methods("GET")

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", router))
}
