package main

import (
	"github.com/bino7/bubbo/bubbo"
	"io/ioutil"
	"crypto/rsa"
	"crypto/rand"
	"log"
	"crypto/x509"
	"fmt"
	"os"
)

var key *rsa.PrivateKey

func main() {
	log,err:= os.OpenFile("log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err!=nil {
		panic(err)
	}
	bubbo.Run(key,log)
}

func init(){
	if keyBytes, err := ioutil.ReadFile("key");err!=nil{
		log.Println("read rsa key from file failed,generate new key")
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err!=nil{
			panic("generate new key failed")
		}
		keyBytes:=x509.MarshalPKCS1PrivateKey(key)
		if err:=ioutil.WriteFile("key",keyBytes,0644);err!=nil{
			panic(fmt.Sprint("write key file failed",err))
		}
	}else{
		key,err=x509.ParsePKCS1PrivateKey(keyBytes)
		if err!=nil{
			panic(fmt.Sprint("parse key from file failed",err))
		}
	}
}