package bubbo

import (
	"net/http"
	"log"
	"github.com/dgrijalva/jwt-go"
	"time"
	"github.com/gorilla/websocket"
	"sync"
	"io"
	"net"
)

func isCloseable(err error) bool {
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		return true
	}

	switch err {
	case io.EOF, io.ErrClosedPipe:
		return true
	}

	return false
}

type peer struct {
	uuid       string
	username   string
	conn       *websocket.Conn
	lastAccess time.Time
	done       chan bool
	handlers   map[string]EventHandler
	mutex      sync.Mutex
	timeout    time.Duration
	in         chan *Event
	out        chan *Event
}

func newPeer() *peer {
	return &peer{
		uuid:"",
		username:"",
		conn:nil,
		lastAccess:time.Now(),
		done:make(chan bool),
		handlers:make(map[string]EventHandler),
		timeout:5 * time.Minute,
		in:make(chan *Event),
		out:make(chan *Event),
	}
}

func (p *peer)ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ok, token := authenticate(w, r)
	if !ok {
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	p.uuid = claims["UUID"].(string)
	p.username = claims["Username"].(string)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	p.conn = conn

	p.On("profile", p.profile)
	p.On("userPeers", p.userPeers)
	p.On("updateUserProfile", p.updateUserProfile)
	p.On("icecandidate", p.connectRemote)
	p.On("offer", p.connectRemote)
	p.On("answer", p.connectRemote)
	/*p.On("media",p.media)
	p.On("addMedia",p.addMedia)
	p.On("removeMedia",p.removeMedia)*/

	peers.add(p)

	go p.readMessage()

	for {
		select {
		case event := <-p.Out():
			log.Println("peer:", p.uuid, "out put", event)
		case <-p.done:
			conn.Close()
			return
		case event := <-p.in:
			group := event.Group
			if event.Type == "err" {
				p.writeMessage(event)
			}
			handler := p.handlers[event.Type]
			for handler != nil {
				event, err := handler(event)
				if err != nil {
					p.writeMessage(errorEvent(ErrBadMsg))
					break
				}
				if event != nil {
					event.Group = group
					handler = p.handlers[event.Type]
				}
			}

			if event != nil {
				p.out <- event
			}
		}
	}
}

func (p *peer)readMessage() {
	for {
		if p.isDone() {
			return
		}
		deadline := time.Now().Add(p.timeout)
		p.conn.SetReadDeadline(deadline)
		event := new(Event)
		err := p.conn.ReadJSON(event)
		if err == io.ErrUnexpectedEOF {
			p.in <- errorEvent(ErrBadMsg)
		} else if isCloseable(err) {
			log.Println("connection closed prematurely: %v", err)
			p.close()
			break
		} else {
			log.Println("failed to read message: %v", err)
		}

		if err == nil {
			p.in <- event
		}
	}
}

func (p *peer)writeMessage(event *Event) {
	deadline := time.Now().Add(p.timeout)
	p.conn.SetWriteDeadline(deadline)
	err := p.conn.WriteJSON(event)
	if isCloseable(err) {
		log.Println("connection closed prematurely: %v", err)
		p.close()
	} else {
		log.Println("failed to write message: %v", err)
	}
	return
}

func (p *peer)On(eventName string, handler EventHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.handlers[eventName] = handler
}

func (p *peer)In() (chan <- *Event) {
	return p.in
}

func (p *peer)Out() (<- chan *Event) {
	return p.out
}

func (p *peer)isDone() {
	select {
	case <-p.done:
		return true
	default:
	}
	return false
}

func (p *peer)profile(eve *Event) (event *Event, err error) {
	u, err := user(p.username)

	if u != nil {
		version := eve.Detail["version"]
		profile := u.profile()
		if profile.Version > version {
			event = &Event{
				Type:"profile",
				Detail:Detail{
					"Username":profile.Username,
					"Title"    :profile.Title,
					"Desc"    :profile.Desc,
					"Email"    :profile.Email,
					"Avatar"   :profile.Avatar,
					"Version"  :profile.Version,
				},
			}
		} else {
			event = &Event{
				Type:"profile",
				Detail:Detail{
					"Version":profile.Version,
				},
			}
		}

		p.writeMessage(event)
	}

	return
}

func (p *peer)updateUserProfile(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"]
	if username != p.username {
		err = ErrorForbidden
		return
	}
	user, _ := user(username)
	user.Username = eve.Detail["username"]
	user.Title = eve.Detail["title"]
	user.Desc = eve.Detail["desc"]
	user.Email = eve.Detail["email"]
	user.Avatar = eve.Detail["avatar"]
	err=updateUser(user)
	return
}

func (p *peer)connectRemote(eve *Event) (event *Event, err error) {
	uuid := eve.Detail["remote"].(string)
	if uuid == p.uuid {
		p.writeMessage(eve)
		return
	}

	remote := peers.getWithId(uuid)
	if remote == nil {
		p.writeMessage(ErrRemoteNotFound)
		return
	}

	remote.In() <- eve
	return
}

func (p *peer)updateUserInfo(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"]
	if username != p.username {
		err = ErrorForbidden
		return
	}

	u, err := user(username)
	if err != nil {
		return err
	}

	u.Username = username
	u.Email = eve.Detail["email"]
	u.Avatar = eve.Detail["avatar"]

	err = updateUser(u)
	if err != nil {
		log.Println(err)
		err = ErrUpdateUserInfoFailed
		return
	}

	event = &Event{
		Type:"updated",
		Detail:{"version":u.Version},
	}

	return
}

func (p *peer)userInfo(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"]
	version := eve.Detail["version"]
	u, err := user(username)
	if err != nil {
		return err
	}
	if u.Version > version {

		event = &Event{
			Type:"userInfo",
			Detail:{"username":u.Username,
				"email":u.Email,
				"avatar":u.Avatar,
				"version":u.Version,
			},
		}
	} else {
		event = &Event{Type:"unmodified"}
	}
	return
}
func (p *peer)userPeers(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"]
	userPeers := peers.getWithUsername(username)
	if userPeers == nil || len(userPeers) == 0 {
		event = &Event{Type:"offline", Detail:{}}
	} else {
		ps := make([]*peer, len(userPeers))
		for i, p := range userPeers {
			ps[i] = p.uuid
		}
		event = &Event{Type:"userPeers", Detail:{"peers":ps}}
	}

	return
}

func (p *peer)timeout(eve *Event) (event *Event, err error) {
	p.writeMessage(eve)
	p.close()
	return
}

func (p *peer) close() {
	peers.remove(p)
	p.done <- true
	p.conn.Close()
}