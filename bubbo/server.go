package bubbo

import (
	"github.com/gorilla/websocket"
	"net/http"
	"github.com/rs/cors"
	"crypto/rsa"
	"os"
	"log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:func(r *http.Request) bool {
		return true
	},
}

var frontendServer="https://bubbo.cn"

var key *rsa.PrivateKey

func Run(rsakey *rsa.PrivateKey,logFile *os.File){
	initStore()

	key=rsakey
	log.SetOutput(logFile)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth",auth)
	mux.HandleFunc("/register",register)
	mux.HandleFunc("/peer",peerConnect)

	c := cors.New(cors.Options{
		AllowedOrigins:     []string{frontendServer},
		AllowedMethods:     []string{"POST", "GET", "PUT", "DELETE"},
		AllowedHeaders:     []string{"Content-Type","Content-Range","Content-Disposition",
						"Accept","Authorization","Set-Cookie",},
		ExposedHeaders:     []string{"Content-Type","Content-Range","Content-Disposition",
						"Accept","Authorization","Set-Cookie",},
		AllowCredentials:   true,
	})

	http.ListenAndServe(":3000", c.Handler(mux))
}

