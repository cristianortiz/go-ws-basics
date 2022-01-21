package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"

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

//WebSocketconnection is wrapper for gorilla/websocket
type WebSocketConnection struct {
	*websocket.Conn
}

//WsJsonResponse is struct to define the response sent back from websocket  as json
type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}

//WsPayload is the struct for the info received from frontend connected client
type WsPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

//go routine to send back to the frontend the payload who ListenForWS, go routine,
//has been upload through channel, is always listening that channel, also broadcast the payload to all connected clients
func ListenToWsChannel() {
	var response WsJsonResponse
	//always listen the channel
	for {
		//get the payload from channel
		e := <-wsChan

		switch e.Action {
		//if a connected client type their username in frontend input and press enter key
		//response is the list of all connected users
		case "username":
			//get a list of all users and send back to frontendt via broadcast
			clients[e.Conn] = e.Username //adding the client username to clients map
			//get the list of clients username
			users := getUserList()
			response.Action = "list_users"
			response.ConnectedUsers = users
			//send the response to all clients
			broadcastToAll(response)
			//if in frontend a client user left the app page, their username is deleted
		case "left_app":
			response.Action = "list_users"
			delete(clients, e.Conn)
			users := getUserList()
			response.ConnectedUsers = users
			//send the response to all clients
			broadcastToAll(response)

		case "broadcast":
			response.Action = "broadcast"
			//create the response data to show on frontend chatbox
			response.Message = fmt.Sprintf("<strong>%s</strong>: %s", e.Username, e.Message)
			broadcastToAll(response)
		}

	}

}

//getUserList loop to map clients and returns slice of strings with the username of all connected clients
func getUserList() []string {
	var usersList []string
	for _, x := range clients {
		//avoid to add connected client who is not write their username yet in app frontend
		if x != "" {
			usersList = append(usersList, x)
		}
	}
	sort.Strings(usersList)
	return usersList
}

//broadcastToALL broadcast the payload received from a client to all connected clients
func broadcastToAll(response WsJsonResponse) {
	for client := range clients {
		err := client.WriteJSON(response)
		//error must be  a disconnected client from the app so deleted it
		if err != nil {
			log.Println("webSocket error")
			_ = client.Close()
			delete(clients, client)
		}

	}
}

//WsEndpoint  get the http requests and response from client tryng to connect, and upgrade it to websocket protocol
//also call ListenToWs() to  process the connected client payloads
func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	//transform the incoming http request and response into websocket protocol
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
	}
	//if there is no error, a client is connected though ws protocol
	log.Println("Client connected to endpoint")
	//set a response msg when a client is connected to ws
	var response WsJsonResponse
	response.Message = `<em><small>Connected to server</small></em>`

	//incoming connection to WsEndpoint is a client of WebSocketConnection type
	// and must be added to clients map
	conn := WebSocketConnection{Conn: ws}
	//temporary empty entry in clients map
	clients[conn] = ""
	//send back the response as JSON
	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
	}
	//goroutine listen for a connected clients payload
	go ListenForWS(&conn)
}

//ListenForWS is a goroutine to listen for incoming payload and send it through channel
func ListenForWS(conn *WebSocketConnection) {
	//if the goroutines is stopped by panic log the error
	defer func() {
		r := recover()
		if r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))

		}
	}()
	// REC: WsPayload is received from connected client
	var payload WsPayload

	//listen eternally for an incoming payload
	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			//do nothing if there ir no error in incomming connections

		} else {
			payload.Conn = *conn
			//send the payload through channel
			wsChan <- payload
		}
	}

}

//renderPage
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
