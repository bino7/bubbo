package bubbo

import (
	"net/http"
	"log"
	"time"
	"github.com/gorilla/websocket"
	"sync"
	"io"
	"net"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

func isCloseable(err error) bool {
	if neterr, ok := err.(net.Error); ok && neterr.Temporary() {
		return false
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
	in         chan *Event
	out        chan *Event
}

func newPeer(uuid string) *peer {
	return &peer{
		uuid:uuid,
		username:"",
		conn:nil,
		lastAccess:time.Now(),
		done:make(chan bool),
		handlers:make(map[string]EventHandler),
		in:make(chan *Event, 1024),
		out:make(chan *Event, 1024),
	}
}

func peerConnect(w http.ResponseWriter, r *http.Request) {
	/*ok, token := authenticate(w, r)
	if !ok {
		upgrader.ReturnError(w, r, http.StatusForbidden, "Forbidden")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	claims := token.Claims.(jwt.MapClaims)
	p.uuid = claims["UUID"].(string)
	p.username = claims["Username"].(string)*/
	r.ParseForm()
	uuid := r.Form.Get("uuid")
	if uuid == "" {
		upgrader.Error(w, r, http.StatusForbidden, ErrorForbidden)
		return
	}
	p := newPeer(uuid)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	conn.SetCloseHandler(func(code int, text string) error {
		log.Println(p.uuid, "on close", "code:", code, "text:", text);
		p.close()
		return nil
	})

	p.conn = conn
	log.Println(p.uuid, "connected")

	p.On("profile", p.profile)
	p.On("peers", p.peers)
	p.On("updateProfile", p.updateProfile)
	p.On("signal", p.signal)
	p.On("following", p.following)
	p.On("followers", p.followers)
	p.On("lookup", p.lookup)
	p.On("invitation", p.invitation)

	peers.add(p)

	go p.readMessage()

	ticker := time.NewTicker(pingPeriod)
	for {
		select {
		case event := <-p.out:
			log.Println("peer:", p.uuid, "out put", event)
		case <-p.done:
			log.Println("peer is done")
			return
		case event := <-p.in:
			id := event.Id
			group := event.Group
			_type := event.Type
			if _type == "err" {
				p.writeMessage(event)
				break
			}

			handler := p.handlers[_type]
			for handler != nil {
				event, err := handler(event)
				if err != nil {
					p.writeMessage(errorEvent(group, _type, err))
					break
				}
				handler = nil
				if event != nil {
					event.Id = id
					event.Group = group
					handler = p.handlers[event.Type]
				}
			}

			if event != nil {
				p.out <- event
			}
		case <-ticker.C:
			p.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := p.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println("ping", err)
				return
			}
		}
	}
}

func (p *peer)readMessage() {
	p.conn.SetReadDeadline(time.Now().Add(pongWait))
	p.conn.SetPongHandler(func(string) error {
		p.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil
	})
	for {
		if p.isDone() {
			return
		}
		event := new(Event)
		err := p.conn.ReadJSON(event)
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				p.in <- errorEvent("*", "badMsg", ErrBadMsg)
			} else if isCloseable(err) {
				log.Println(p.uuid, "connection closed prematurely:", err)
			} else {
				log.Println(p.uuid, "failed to read message:", err)
			}
			p.close()
		} else {
			log.Println("read", event.Type)
			p.in <- event
			log.Println("puted event")
		}

	}
}

func (p *peer)writeMessage(event *Event) {
	if p.isDone() {
		return
	}
	p.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := p.conn.WriteJSON(event)
	if err != nil {
		if isCloseable(err) {
			log.Println(p.uuid, "connection closed prematurely:", err)
			p.close()
		} else {
			log.Println("failed to write message:", err)
		}
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

func (p *peer)isDone() bool {
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
		version := int64(eve.Detail["version"].(float64))
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

func (p *peer)updateProfile(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"].(string)
	if username != p.username {
		err = ErrorForbidden
		return
	}
	user, _ := user(username)
	user.Username = eve.Detail["username"].(string)
	user.Title = eve.Detail["title"].(string)
	user.Desc = eve.Detail["desc"].(string)
	user.Email = eve.Detail["email"].(string)
	user.Avatar = eve.Detail["avatar"].(string)
	ver, err := updateUser(user)
	if err != nil {
		return
	}
	event = &Event{Type:"userPeers", Detail:Detail{"result":"success", "version":ver}}

	return
}

func (p *peer)signal(eve *Event) (event *Event, err error) {
	from := eve.Detail["from"].(string)
	to := eve.Detail["to"].(string)
	log.Println(from, "?-->", to)
	if to == p.uuid {
		p.writeMessage(eve)
		return
	}

	remote := peers.getWithId(to)
	if remote == nil {
		p.writeMessage(errorEvent(eve.Group, eve.Type, ErrRemoteNotFound))
		return
	}

	log.Println(from, "-->", to)
	remote.In() <- eve
	return
}

func (p *peer)peers(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"].(string)
	userPeers := peers.getWithUsername(username)
	if userPeers == nil || len(userPeers) == 0 {
		event = &Event{Type:"offline", Detail:Detail{}}
	} else {
		ps := make([]string, len(userPeers))
		for i, p := range userPeers {
			ps[i] = p.uuid
		}
		event = &Event{Type:"peers", Detail:Detail{"peers":ps}}
	}

	return
}

func (p *peer)following(eve *Event) (event *Event, err error) {
	version := int64(eve.Detail["version"].(float64))
	u, err := user(p.username);
	if err != nil {
		return
	}

	f, err := u.following(version)
	if err != nil {
		return
	}

	event = &Event{Type:"following", Detail:Detail{"following":f}}

	return
}

func (p *peer)followers(eve *Event) (event *Event, err error) {
	version := int64(eve.Detail["version"].(float64))
	u, err := user(p.username);
	if err != nil {
		return
	}

	f, err := u.followers(version)
	if err != nil {
		return
	}

	event = &Event{Type:"followers", Detail:Detail{"followers":f}}

	return
}

func (p *peer)lookup(e *Event) (event *Event, err error) {
	log.Println("lookup", e)
	uuid := e.Detail["uuid"].(string)
	if peers.peerEntry[uuid] != nil {
		e.Detail["result"] = true
		p.writeMessage(e)
	} else {
		err = ErrRemoteNotFound
	}
	return
}

func (p *peer)invitation(e *Event) (event *Event, err error) {
	from := e.Detail["from"].(string)
	to := e.Detail["to"].(string)
	reply := e.Detail["reply"]

	if (reply == nil) {
		if to == p.uuid {
			p.writeMessage(e)
			return
		}
		remote := peers.getWithId(to)
		if remote == nil {
			p.writeMessage(errorEvent(e.Group, e.Type, ErrRemoteNotFound))
			return
		}

		remote.In() <- e
	} else {
		if from == p.uuid {
			p.writeMessage(e)
			return
		}
		remote := peers.getWithId(from)
		if remote == nil {
			p.writeMessage(errorEvent(e.Group, e.Type, ErrRemoteNotFound))
			return
		}

		remote.In() <- e
	}

	return
}

func (p *peer) close() {
	if p.isDone() {
		return
	}
	peers.remove(p)
	p.done <- true
	p.conn.Close()
	log.Println(p.uuid, "closed")
}