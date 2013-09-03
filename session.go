package easygo

import (
	//	"../../apps"
	"github.com/matyhtf/easygo/php"
	"bytes"
	"log"
	"net/http"
	"os"
	"time"
	//"fmt"
	"encoding/gob"
	"io/ioutil"
	"runtime"
//	"strings"
	"syscall"
)

type SessionType struct {
	Id string
	Change bool
	Store *SessionItem
}

type SessionItem struct {
	Data   map[string]string
	Expire int64
}

var session_map map[string]*SessionItem = make(map[string]*SessionItem, 100)

func NewSession(req *http.Request, resp http.ResponseWriter) *SessionType {
	s := new(SessionType)
	s.Change = false
	cookie, err := req.Cookie(Server.SessionKey)
	if err != nil {
		s.Id, err = php.Uniqid()
		if err != nil {
			panic(err)
		}
	} else {
		s.Id = cookie.Value
	}
	session := session_map[s.Id]
	if session == nil {
		session = new(SessionItem)
		session.Data = make(map[string]string, 10)

		var buf bytes.Buffer
		b, err := ioutil.ReadFile(Server.SessionDir + s.Id)
		if err == nil {
			de := gob.NewDecoder(&buf)
			buf.Write(b)
			de.Decode(&session)
		}
		session_map[s.Id] = session
	}
	s.Store = session
	expire := time.Now().Add(time.Duration(Server.SessionLifetime) * time.Second)
	s.Store.Expire = expire.Unix()

	http.SetCookie(resp, &http.Cookie{
		Name:    Server.SessionKey,
		Value:   s.Id,
		Path:    "/",
		Expires: expire,
	})
	return s
} 

func (s *SessionType) Init() {
	err := syscall.Access(Server.SessionDir, syscall.O_RDONLY)
	if err != nil {
		os.Mkdir(Server.SessionDir, 0755)
	}
}

func (s *SessionType) Set(key string, value string) {
	s.Store.Data[key] = value
	s.Change = true
}

func (s *SessionType) Get(key string) string {
	return s.Store.Data[key]
}

func (s *SessionType) Del(key string) {
	delete(s.Store.Data, key)
	s.Change = true
}

func (s *SessionType) Save() {
	if len(session_map[s.Id].Data) == 0 || !s.Change{
		return
	}
	var buf bytes.Buffer
	en := gob.NewEncoder(&buf)
	err := en.Encode(session_map[s.Id])
	if err != nil {
		log.Fatal("encode error:", err)
	}
	err = ioutil.WriteFile(Server.SessionDir+s.Id, buf.Bytes(), 0755)
	if err != nil {
		log.Fatal("write file error:", err)
	}
}

func Session_CheckExpire() {
	timer := time.NewTicker(60 * time.Second)
	for {
		<-timer.C
		//log.Println("session_check_expire")
		now := time.Now().Unix()
		for i, session := range session_map {
			if session.Expire < now {
				delete(session_map, i)
				os.Remove(Server.SessionDir + i)
				runtime.Gosched()
			}
		}
	}
}
