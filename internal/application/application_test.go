package application_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saykoooo/calc_go/internal/application"
	"github.com/saykoooo/calc_go/pkg/calculation"
)

// Тест статус 200
func TestCalcHandler_Success(t *testing.T) {
	reqBody := &application.Request{Expression: "3 + 4"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(application.CalcHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp application.RespOk
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if resp.Result != "7.000000" {
		t.Errorf("handler returned unexpected body: got %v want %v", resp.Result, "7.000000")
	}
}

// Тест статус 405
func TestCalcHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calculate", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(application.CalcHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

// Тест статус 400
func TestCalcHandler_BadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(application.CalcHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// Тест статус 422
func TestCalcHandler_InvalidExpression(t *testing.T) {
	reqBody := &application.Request{Expression: "3 / 0"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(application.CalcHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnprocessableEntity {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnprocessableEntity)
	}

	var resp application.RespError
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if resp.Error != calculation.ErrInvalidExpression.Error() {
		t.Errorf("handler returned unexpected error message: got %v want %v", resp.Error, calculation.ErrInvalidExpression.Error())
	}
}

// Тест статус 404
func TestNotFoundHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(application.NotFoundHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}
