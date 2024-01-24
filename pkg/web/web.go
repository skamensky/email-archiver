package web

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/skamensky/email-archiver/pkg/database"
	"github.com/skamensky/email-archiver/pkg/email"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/options"
	"github.com/skamensky/email-archiver/pkg/utils"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

//go:embed static
var staticDir embed.FS
var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{}          // use default options
var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan string)            // broadcast channel
var mutex = &sync.Mutex{}                    // mutex to protect clients

var idMutex = &sync.Mutex{}
var lastId int64 = 0

func messageId() int64 {
	idMutex.Lock()
	defer idMutex.Unlock()
	lastId++
	return lastId
}

// handle incoming websocket messages
func handleMessages() {
	for {
		msg := <-broadcast

		data := map[string]interface{}{}
		data["data"] = msg
		data["id"] = messageId()
		data["numClients"] = len(clients)
		asJson := utils.MustJSON(data)
		mutex.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, []byte(asJson))
			if err != nil {
				log.Printf("websocket write error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}

func apiDec(handler func(http.ResponseWriter, *http.Request) (int, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		type postErrorResponse struct {
			Error     string `json:"error"`
			ErrorCode int    `json:"errorCode"`
		}

		respCode, err := handler(w, r)
		if err != nil {
			resp := postErrorResponse{Error: err.Error(), ErrorCode: respCode}
			respJson, err := json.Marshal(resp)
			if err != nil {
				log.Printf("error marshalling error response: %v", err)
				http.Error(w, "error marshalling error response", http.StatusInternalServerError)
				return
			} else {
				http.Error(w, string(respJson), respCode)
			}
		}
	}
}

func allowedMethodsDec(handler func(http.ResponseWriter, *http.Request), methods ...string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := false
		for _, m := range methods {
			if r.Method == m {
				allowed = true
			}
		}
		if allowed {
			handler(w, r)
			return
		} else {
			http.Error(w, fmt.Sprintf("only %v allowed", methods), http.StatusMethodNotAllowed)
		}
	}
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	// Register client
	mutex.Lock()
	clients[c] = true
	mutex.Unlock()

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			mutex.Lock()
			delete(clients, c)
			mutex.Unlock()
			if !websocket.IsCloseError(err, websocket.CloseGoingAway) {
				log.Printf("read error: %v", err)
			}
			break
		}
	}
}

func getOptions(w http.ResponseWriter, r *http.Request) (int, error) {
	ops, err := options.New()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	_, err = w.Write([]byte(utils.MustJSON(ops)))
	return http.StatusOK, err
}

/*
type SQLWhereOperator string

type SQLWhereCondition struct {
	Column   string
	Value    string
	Operator SQLWhereOperator
	Extra    string
}
*/

func getEmails(w http.ResponseWriter, r *http.Request) (int, error) {
	// parse array of conditions from body:
	// read json frombody:
	type postBody struct {
		Conditions []models.SQLWhereCondition `json:"conditions"`
	}

	type postResponse struct {
		Emails []*email.Email `json:"emails"`
	}

	type postErrorResponse struct {
		Error string `json:"error"`
	}

	var body postBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return http.StatusBadRequest, utils.JoinErrors("error decoding json", err)
	}

	if len(body.Conditions) == 0 {
		return http.StatusBadRequest, errors.New("no conditions provided")
	}

	emailsInt, err := database.GetDatabase().GetEmails(body.Conditions)
	emails := []*email.Email{}
	for _, e := range emailsInt {
		emails = append(emails, e.(*email.Email))
	}

	response := postResponse{emails}
	respJson, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		_, err = w.Write(respJson)
		utils.PanicIfError(err)
	}

	return http.StatusOK, nil
}

func Start() error {

	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/api/options", allowedMethodsDec(apiDec(getOptions), http.MethodGet, http.MethodOptions))
	http.HandleFunc("/api/emails", allowedMethodsDec(apiDec(getEmails), http.MethodPost, http.MethodOptions))

	go handleMessages()

	go randomWriter()
	content, err := fs.Sub(staticDir, "static")
	if err != nil {
		return utils.JoinErrors("failed to load static content", err)
	}
	http.Handle("/", http.FileServer(http.FS(content)))
	fmt.Printf("Starting server at %s\n", *addr)
	return http.ListenAndServe(*addr, nil)
}
func randomWriter() {
	for {
		waited := rand.Intn(10)

		time.Sleep(time.Duration(waited) * time.Second)
		broadcast <- "Waited " + strconv.Itoa(waited) + " seconds"
	}
}
