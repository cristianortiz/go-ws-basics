package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CloudyKit/jet/v6"
	"github.com/gorilla/websocket"
)

//channel to comunicate goroutines, only accepts WsPaylods types
var wsChan = make(chan WsPayload)

//map of clients, every client connected have their own WebSocketConnection
var clients = make(map[WebSocketConnection]string)

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./html"),
	jet.InDevelopmentMode(),
)

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

//Home functions render the main webpage
func Home(w http.ResponseWriter, r *http.Request) {
	err := renderPage(w, "home.jet", nil)
	if err != nil {
		log.Println(err)
	}
}

//WebSocketconnection is wrapper for gorilla.conn
type WebSocketConnection struct {
	*websocket.Conn
}

//WsJsonResponse is struct to define the response sent back from websocket  as json
type WsJsonResponse struct {
	Action      string `json:"action"`
	Message     string `json:"message"`
	MessageType string `json:"message_type"`
}

//WsPayload is the struct for the info received from frontend
type WsPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

//func to listen and response back to frontend through channel
func ListenToWsChannel() {
	var response WsJsonResponse
	//always listen the channel
	for {
		//get the payload from channel
		e := <-wsChan
		//set the response
		response.Action = "Got Here"
		response.Message = fmt.Sprintf("Some messagem and action was %s", e.Action)
		//send the response to all clients
		broadcastToAll(response)
	}

}

//func to broadcast the response to all clients
func broadcastToAll(response WsJsonResponse) {
	for client := range clients {
		err := client.WriteJSON(response)
		//error mus be due the clients disconnect form the app so deleted it
		if err != nil {
			log.Println("webSocket error")
			_ = client.Close()
			delete(clients, client)
		}

	}
}

//WsEndpoint function to get http connection and upgrade it to a websocket conn
func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
	}

	log.Println("Client connected to endpoint")
	//send back a response
	var response WsJsonResponse
	response.Message = `<em><small>Connected to server</small></em>`

	//incoming connection to WsEndpoint is a client of WebSocketConnection type
	// and must be added to clients map
	conn := WebSocketConnection{Conn: ws}
	//temporary empry entry in clients
	clients[conn] = ""

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
	}
	//goroutine to always listen for a payload
	go ListenForWS(&conn)
}

func ListenForWS(conn *WebSocketConnection) {
	//if the goroutines is stopped by panic log the error
	defer func() {
		r := recover()
		if r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))

		}
	}()
	// REC: WsPayload is received from frontend
	var payload WsPayload

	//listen eternally for an incoming payload
	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			//do nothing if there ir no error in incomming connections

		} else {
			//pointer to
			payload.Conn = *conn
			//send the payload throuh channel
			wsChan <- payload
		}
	}

}

func renderPage(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	view, err := views.GetTemplate(tmpl)
	if err != nil {
		log.Println(err)
		return err
	}
	err = view.Execute(w, data, nil)

	if err != nil {
		log.Println(err)
		return err

	}
	return nil
}
