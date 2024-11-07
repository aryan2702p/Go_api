package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type OllamaClient struct {
    baseURL string
}

type OllamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
}

type OllamaResponse struct {
    Response string `json:"response"`
}

func NewOllamaClient(baseURL string) *OllamaClient {
    return &OllamaClient{baseURL: baseURL}
}

func (c *OllamaClient) GenerateStudentSummary(student Student) (string, error) {
    prompt := fmt.Sprintf(
        "Generate a brief summary of this student:\nName: %s\nAge: %d\nEmail: %s",
        student.Name,
        student.Age,
        student.Email,
    )

    reqBody := OllamaRequest{
        Model:  "llama2",
        Prompt: prompt,
    }

    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return "", err
    }

    resp, err := http.Post(
        c.baseURL+"/api/generate",
        "application/json",
        bytes.NewBuffer(jsonBody),
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var ollamaResp OllamaResponse
    if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
        return "", err
    }

    return ollamaResp.Response, nil
}
