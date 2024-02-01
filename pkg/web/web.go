package web

import (
	"embed"
	"encoding/json"
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
	"net/http"
	"sync"
)

//go:embed frontend/build
var frontendBuildDir embed.FS
var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{}
var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan interface{})       // broadcast channel
var mutex = &sync.Mutex{}                    // mutex to protect clients

var idMutex = &sync.Mutex{}
var lastId int64 = 0

var pool models.ClientPool

type successResponse struct {
	Success bool `json:"success"`
}

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
		asJson := utils.MustJSON(data)
		mutex.Lock()
		// todo: only send to clients that are subscribed to this data
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
		// "host:port" [METHOD] path?query"

		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		// get client ip:

		utils.DebugPrintln(fmt.Sprintf("%s [%s] %s", r.RemoteAddr, r.Method, path))
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

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	// allow all origins (mainly for dev)
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade error:", err)
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
		SqlQuery string `json:"sqlQuery"`
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

	if body.SqlQuery == "" {
		return http.StatusBadRequest, utils.JoinErrors("sqlQuery is required", nil)
	}
	emailsMod, err := database.GetDatabase().GetEmails(body.SqlQuery)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error getting emails", err)
	}

	emails := []*email.Email{}
	for _, e := range emailsMod {
		emails = append(emails, e.(*email.Email))

	}

	response := postResponse{emails}
	respJson, err := json.Marshal(response)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error marshalling response", err)
	} else {
		_, err = w.Write(respJson)
		utils.PanicIfError(err)
	}

	return http.StatusOK, nil
}

func searchEmails(w http.ResponseWriter, r *http.Request) (int, error) {
	// parse array of conditions from body:
	// read json frombody:
	type postBody struct {
		SearchQuery string `json:"searchQuery"`
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

	emailsMod, err := database.GetDatabase().FullTextSearch(body.SearchQuery)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error getting emails", err)
	}

	emails := []*email.Email{}
	for _, e := range emailsMod {
		emails = append(emails, e.(*email.Email))

	}

	response := postResponse{emails}
	respJson, err := json.Marshal(response)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error marshalling response", err)
	} else {
		_, err = w.Write(respJson)
		utils.PanicIfError(err)
	}

	return http.StatusOK, nil
}

func getMailboxes(w http.ResponseWriter, r *http.Request) (int, error) {
	mailboxes, err := pool.ListMailboxes()
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error getting mailboxes", err)
	}
	type postResponse struct {
		Mailboxes []models.MailboxRecord `json:"mailboxes"`
	}

	response := postResponse{}
	for _, m := range mailboxes {
		response.Mailboxes = append(response.Mailboxes, m.MailboxRecord())
	}

	respJson, err := json.Marshal(response)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error marshalling response", err)
	} else {
		_, err = w.Write(respJson)
		utils.PanicIfError(err)
	}

	return http.StatusOK, nil
}

func ImapEventHandler(event *models.MailboxEvent) {
	broadcast <- event
}

func syncMailboxes(w http.ResponseWriter, r *http.Request) (int, error) {

	type postBody struct {
		Mailboxes []string `json:"mailboxes"`
	}

	var body postBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return http.StatusBadRequest, utils.JoinErrors("error decoding json", err)
	}

	mailboxesRequestedSet := utils.NewSet(body.Mailboxes)
	mailboxes, err := pool.ListMailboxes()
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error getting mailboxes", err)
	}
	mailboxesToUse := []models.Mailbox{}
	for _, m := range mailboxes {
		if mailboxesRequestedSet.Contains(m.Name()) {
			mailboxesToUse = append(mailboxesToUse, m)
		}
	}

	err = pool.DownloadMailboxes(mailboxesToUse)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error syncing mailboxes", err)
	}

	response := successResponse{true}
	respJson, err := json.Marshal(response)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error marshalling response", err)
	} else {
		_, err = w.Write(respJson)
		utils.PanicIfError(err)
	}

	return http.StatusOK, nil

}

func setFrontEndState(w http.ResponseWriter, r *http.Request) (int, error) {
	type postBody struct {
		State string `json:"state"`
	}

	var body postBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return http.StatusBadRequest, utils.JoinErrors("error decoding json", err)
	}

	//get db
	db := database.GetDatabase()
	err = db.SetFrontendState(body.State)

	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error setting frontend state", err)
	}

	response := successResponse{true}
	respJson, err := json.Marshal(response)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error marshalling response", err)
	} else {
		_, err = w.Write(respJson)
		utils.PanicIfError(err)
	}
	return http.StatusOK, nil
}

func getFrontEndState(w http.ResponseWriter, r *http.Request) (int, error) {
	db := database.GetDatabase()
	state, err := db.GetFrontendState()
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error getting frontend state", err)
	}

	type getResponse struct {
		State string `json:"state"`
	}

	response := getResponse{state}
	respJson, err := json.Marshal(response)
	if err != nil {
		return http.StatusInternalServerError, utils.JoinErrors("error marshalling response", err)
	} else {
		_, err = w.Write(respJson)
		utils.PanicIfError(err)
	}
	return http.StatusOK, nil
}

func Start(imapConnPool models.ClientPool) error {
	pool = imapConnPool

	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/ws", websocketHandler)
	http.HandleFunc("/api/options", allowedMethodsDec(apiDec(getOptions), http.MethodGet, http.MethodOptions))
	http.HandleFunc("/api/emails", allowedMethodsDec(apiDec(getEmails), http.MethodPost, http.MethodOptions))
	http.HandleFunc("/api/mailboxes", allowedMethodsDec(apiDec(getMailboxes), http.MethodGet, http.MethodOptions))
	http.HandleFunc("/api/sync", allowedMethodsDec(apiDec(syncMailboxes), http.MethodPost, http.MethodOptions))
	http.HandleFunc("/api/search", allowedMethodsDec(apiDec(searchEmails), http.MethodPost, http.MethodOptions))
	http.HandleFunc("/api/set_frontend_state", allowedMethodsDec(apiDec(setFrontEndState), http.MethodPost, http.MethodOptions))
	http.HandleFunc("/api/get_frontend_state", allowedMethodsDec(apiDec(getFrontEndState), http.MethodGet, http.MethodOptions))

	go handleMessages()
	// hydrate mailbox cache on startup since it's an operation that takes a while
	go pool.ListMailboxes()

	content, err := fs.Sub(frontendBuildDir, "frontend/build")
	if err != nil {
		return utils.JoinErrors("failed to load static content", err)
	}
	http.Handle("/", http.FileServer(http.FS(content)))

	fmt.Printf("Starting server at %s\n", *addr)
	return http.ListenAndServe(*addr, nil)
}
