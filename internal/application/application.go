package application

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/saykoooo/calc_go/pkg/calculation"
	"log"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Addr string
}

type RespOk struct {
	Result string `json:"result"`
}

type RespError struct {
	Error string `json:"error"`
}

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
		config.Addr = "8080"
	}
	return config
}

type Application struct {
	config *Config
}

func New() *Application {
	return &Application{
		config: ConfigFromEnv(),
	}
}

// Функция запуска приложения
// тут будем чиать введенную строку и после нажатия ENTER писать результат работы программы на экране
// если пользователь ввел exit - то останаваливаем приложение
func (a *Application) Run() error {
	for {
		// читаем выражение для вычисления из командной строки
		log.Println("input expression")
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Println("failed to read expression from console")
		}
		// убираем пробелы, чтобы оставить только вычислемое выражение
		text = strings.TrimSpace(text)
		// выходим, если ввели команду "exit"
		if text == "exit" {
			log.Println("aplication was successfully closed")
			return nil
		}
		//вычисляем выражение
		result, err := calculation.Calc(text)
		if err != nil {
			log.Println(text, " calculation failed wit error: ", err)
		} else {
			log.Println(text, "=", result)
		}
	}
}

type Request struct {
	Expression string `json:"expression"`
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/calculate" {
		http.Error(w, "Bad URL", http.StatusNotFound)
		return
	}
}

func CalcHandler(w http.ResponseWriter, r *http.Request) {
	request := new(Request)
	defer r.Body.Close()
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	result, err := calculation.Calc(request.Expression)
	if err != nil {
		if errors.Is(err, calculation.ErrInvalidExpression) || errors.Is(err, calculation.ErrDivByZero) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			errJsonData, _ := json.Marshal(RespError{Error: err.Error()})
			w.Write(errJsonData)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			errJsonData, _ := json.Marshal(RespError{Error: "Internal server error"})
			w.Write(errJsonData)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		s := fmt.Sprintf("%f", result)
		okJsonData, _ := json.Marshal(RespOk{Result: s})
		w.Write(okJsonData)
	}
}

func (a *Application) RunServer() error {
	mux := http.NewServeMux()
	mux.Handle("/", LoggingMiddleware(http.HandlerFunc(NotFoundHandler)))
	mux.Handle("/api/v1/calculate", LoggingMiddleware(http.HandlerFunc(CalcHandler)))
	log.Printf("Web server run on port: %s\n", a.config.Addr)
	return http.ListenAndServe(":"+a.config.Addr, mux)
}
