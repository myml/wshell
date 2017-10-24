// wshell project main.go
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/asdine/storm"
	"github.com/go-macaron/macaron"
	"github.com/gorilla/websocket"
)

var modes = ssh.TerminalModes{
	ssh.ECHO:          1,     // disable echoing
	ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
}

type Result struct {
	Err    interface{}
	Result interface{}
}
type SSH struct {
	ID       int `storm:"id,increment"`
	Host     string
	User     string
	Title    string
	AuthType string
	Key      string
	Pwd      string
}

var db *storm.DB

func init() {
	var err error
	db, err = storm.Open("db")
	if err != nil {
		log.Panic(err)
	}
}
func main() {
	m := macaron.New()
	m.Use(macaron.Logger())
	m.Use(macaron.Static("html"))
	m.Use(macaron.Renderer(macaron.RenderOptions{
		Delims: macaron.Delims{"{!", "!}"},
	}))
	m.Use(func(ctx *macaron.Context) {
		defer func() {
			err := recover()
			if err != nil {
				log.Println("error:", err)
				ctx.JSON(200, &Result{
					Err: err,
				})
			}
		}()
		ctx.Next()
		if result, ok := ctx.Data["result"]; ok {
			ctx.JSON(200, &Result{
				Result: result,
			})
		}
	})
	m.Get("ssh", func(ctx *macaron.Context) {
		var sList []SSH
		if err := db.All(&sList); err != nil {
			log.Panic(err)
		}
		ctx.Data["result"] = sList
	})
	m.Post("ssh", func(ctx *macaron.Context) {
		b, err := ctx.Req.Body().Bytes()
		if err != nil {
			log.Panic(err)
		}
		var s SSH
		if err = json.Unmarshal(b, &s); err != nil {
			log.Panic(err)
		}
		if err = db.Save(&s); err != nil {
			log.Panic(err)
		}
		log.Println(s)
		ctx.Data["result"] = s.ID
	})
	m.Put("ssh", func(ctx *macaron.Context) {
		b, err := ctx.Req.Body().Bytes()
		if err != nil {
			log.Panic(err)
		}
		var s SSH
		if err = json.Unmarshal(b, &s); err != nil {
			log.Panic(err)
		}
		if err = db.Update(&s); err != nil {
			log.Panic(err)
		}
	})
	m.Delete("ssh/:id", func(ctx *macaron.Context) {
		id := ctx.ParamsInt("id")
		var s SSH
		if err := db.One("ID", id, &s); err != nil {
			log.Panic(err)
		}
		if err := db.DeleteStruct(&s); err != nil {
			log.Panic(err)
		}
	})
	m.Get("user", func(ctx *macaron.Context) {
		u, err := user.Current()
		if err != nil {
			log.Panic(err)
		}
		ctx.Data["result"] = u
	})

	m.Get("ssh/:id", func(ctx *macaron.Context) {
		ws, err := websocket.Upgrade(ctx.Resp, ctx.Req.Request, nil, 1024, 1024)
		if err != nil {
			log.Panic(err)
		}
		defer ws.Close()

		id := ctx.ParamsInt("id")
		log.Println(id)
		var s SSH
		if err = db.One("ID", id, &s); err != nil {
			log.Panic(err)
		}

		conf := &ssh.ClientConfig{
			User:            s.User,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		if s.AuthType == "key" {
			rsa, err := ioutil.ReadFile(s.Key)
			k, err := ssh.ParsePrivateKey(rsa)
			if err != nil {
				log.Panic(err)
			}
			conf.Auth = []ssh.AuthMethod{
				ssh.PublicKeys(k),
			}
		} else {
			conf.Auth = []ssh.AuthMethod{
				ssh.Password(s.Pwd),
			}
		}
		if !strings.Contains(s.Host, ":") {
			s.Host += ":22"
		}
		conn, err := ssh.Dial("tcp", s.Host, conf)
		if err != nil {
			log.Panic(err)
		}
		session, err := conn.NewSession()
		if err != nil {
			log.Panic(err)
		}
		session.RequestPty("xterm", 120, 80, modes)

		in, err := session.StdinPipe()
		if err != nil {
			log.Panic(err)
		}
		out, err := session.StdoutPipe()
		if err != nil {
			log.Panic(err)
		}
		session.Stderr = os.Stdout
		session.Shell()
		go func() {
			var b [512]byte
			for {
				n, err := out.Read(b[:])
				if err != nil {
					log.Println(err)
					return
				}
				ws.WriteMessage(websocket.TextMessage, b[:n])
			}
		}()
		type Message struct {
			Type string
			Cols int
			Rows int
			Data string
		}
		for {
			var msg Message
			if err := ws.ReadJSON(&msg); err != nil {
				log.Panic(err)
			}
			if msg.Type == "data" {
				in.Write([]byte(msg.Data))
			} else if msg.Type == "resize" {
				session.WindowChange(msg.Rows, msg.Cols)
				log.Println(msg)
			}
		}
	})
	m.Run()
}
