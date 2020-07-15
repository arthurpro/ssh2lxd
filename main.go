package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/gorilla/websocket"
	"github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

const appName = "ssh2lxd"

var (
	//deadlineTimeout = 30 * time.Second
	idleTimeout = 180 * time.Second

	flagDebug  = false
	flagListen = ":2222"
	flagHelp   = false
	flagSocket = "/var/snap/lxd/common/lxd/unix.socket"
	flagNoauth = false
	flagGroups = "wheel,lxd"

	allowedGroups []string

	lxdSocket = ""
)

func init() {
	flag.BoolVarP(&flagHelp, "help", "h", flagHelp, "print help")
	flag.BoolVarP(&flagDebug, "debug", "d", flagDebug, "enable debug")
	flag.BoolVarP(&flagNoauth, "noauth", "n", flagDebug, "disable public key auth")
	flag.StringVarP(&flagListen, "listen", "l", flagListen, "listen on :2222 or 127.0.0.1:2222")
	flag.StringVarP(&flagSocket, "socket", "s", flagSocket, "LXD socket or use LXD_SOCKET")
	flag.StringVarP(&flagGroups, "groups", "g", flagGroups, "user must belong to one of the groups to authenticate")
	flag.Parse()

	if flagHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	lxdSocket = os.Getenv("LXD_SOCKET")
	if lxdSocket == "" {
		lxdSocket = flagSocket
	}

	allowedGroups = strings.Split(flagGroups, ",")
}

func main() {
	log.SetOutput(os.Stdout)
	if flagDebug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.ErrorLevel)
	}

	ssh.Handle(func(s ssh.Session) {
		HandleShell(s)
	})

	var authHandler ssh.PublicKeyHandler
	if flagNoauth {
		authHandler = nil
	} else {
		authHandler = handlePubKeyAuth
	}

	if len(allowedGroups) > 0 {
		allowedGroups = append([]string{"0"}, getGroupIds(allowedGroups)...)
	}

	server := &ssh.Server{
		Addr: flagListen,
		//MaxTimeout:  deadlineTimeout,
		IdleTimeout:      idleTimeout,
		Version:          "LXD",
		PublicKeyHandler: authHandler,
	}

	fmt.Printf("Starting %s server on %s, LXD socket %s\n", appName, flagListen, lxdSocket)
	log.Fatal(server.ListenAndServe())
}

func handlePubKeyAuth(ctx ssh.Context, key ssh.PublicKey) bool {
	var user string

	user, _, _ = parseUser(ctx.User())

	osUser, err := getOsUser(user)
	if err != nil {
		return false
	}

	if len(allowedGroups) > 0 {
		userGroups, err := getUserGroups(osUser)
		if err != nil {
			return false
		}
		if !isGroupMatch(allowedGroups, userGroups) {
			log.Infof("Group match failed for user: %s %v", user, userGroups)
			return false
		}
		log.Debugf("Group matched for user: %s %v %v", user, userGroups, allowedGroups)
	}

	keys, _ := getUserAuthKeys(osUser)
	for _, k := range keys {
		pk, _, _, _, err := ssh.ParseAuthorizedKey(k)
		if err != nil {
			log.Debugf("ssh.ParseAuthorizedKey error: %v", err)
			continue
		}
		if ssh.KeysEqual(pk, key) {
			return true
		}
	}

	log.Infof("Auth failed for user: %s", user)
	return false
}

func HandleShell(s ssh.Session) {
	command := s.Command()
	env := make(map[string]string)

	user, container, containerUser := parseUser(s.User())

	log.Infof("User: %s, Container: %s", user, container)

	ptyReq, winCh, isPty := s.Pty()

	lxdDaemon, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		log.Errorf("LXD connection error: %v", err)
		s.Exit(255)
		return
	}
	defer lxdDaemon.Disconnect()

	log.Debugf("Command: %v", s.RawCommand())

	args := []string{}
	for _, v := range s.Environ() {
		k := strings.Split(v, "=")
		env[k[0]] = k[1]
	}
	if ptyReq.Term != "" {
		env["TERM"] = ptyReq.Term
	} else {
		env["TERM"] = "xterm-256color"
	}

	if ssh.AgentRequested(s) {
		l, err := ssh.NewAgentListener()
		if err != nil {
			log.Errorf("ssh.NewAgentListener error: %v", err)
		}
		defer l.Close()
		go ssh.ForwardAgentConnections(l, s)

		device, socket := addProxySocketDevice(lxdDaemon, container, l.Addr().String())
		if device != "" {
			env["SSH_AUTH_SOCK"] = socket
			defer removeProxySocketDevice(lxdDaemon, container, device)
		}
	}

	if len(command) < 1 {
		args = append(args,
			"su",
			"-m", // preserve environment for ssh auth socket to be set
			//"-", // enabling this will ignore previous option
			containerUser,
		)
	}

	args = append(args, command...)
	//args = append(args, s.RawCommand())

	log.Printf("Executing '%v'", strings.Join(args, " "))
	log.Debugf("Pty: %v", isPty)
	log.Debugf("ENV: %v", env)

	var ws *websocket.Conn
	defer func() {
		if ws != nil {
			ws.Close()
		}
	}()
	handler := func(conn *websocket.Conn) {
		ws = conn
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				// pointless
				//log.Errorf("ws.ReadMessage() error: %v", err)
				break
			}
		}
	}

	var execArgs lxd.InstanceExecArgs
	var req api.InstanceExecPost

	if isPty {
		req = api.InstanceExecPost{
			Command:     args,
			WaitForWS:   true,
			Interactive: true,
			Environment: env,
			Width:       ptyReq.Window.Width,
			Height:      ptyReq.Window.Height,
		}

		execArgs = lxd.InstanceExecArgs{
			Stdin:    s,
			Stdout:   s,
			Stderr:   s,
			Control:  handler,
			DataDone: make(chan bool),
		}
	} else {
		req = api.InstanceExecPost{
			Command:     args,
			WaitForWS:   true,
			Interactive: false,
			Environment: env,
			Width:       ptyReq.Window.Width,
			Height:      ptyReq.Window.Height,
		}

		inRead, inWrite := io.Pipe()
		//outRead, outWrite := io.Pipe()
		errRead, errWrite := io.Pipe()

		go func(s ssh.Session, w io.WriteCloser) {
			defer w.Close()
			io.Copy(w, s)
		}(s, inWrite)

		go func(s ssh.Session, e io.ReadCloser) {
			defer e.Close()
			io.Copy(s.Stderr(), e)
		}(s, errRead)

		//go func(s ssh.Session, r io.ReadCloser, e io.ReadCloser){
		//	defer r.Close()
		//	defer e.Close()
		//	b := io.MultiReader(outRead, errRead)
		//	io.Copy(s, b)
		//}(s, outRead, errRead)
		//
		//execArgs := lxd.ContainerExecArgs{
		//	Stdin:    inRead,
		//	Stdout:   outWrite,
		//	Stderr:   errWrite,
		//	Control:  handler,
		//	DataDone: make(chan bool),
		//}

		execArgs = lxd.InstanceExecArgs{
			Stdin:    inRead,
			Stdout:   s,
			Stderr:   errWrite,
			Control:  handler,
			DataDone: make(chan bool),
		}

	}

	op, err := lxdDaemon.ExecInstance(container, req, &execArgs)
	if err != nil {
		log.Errorf("lxdDaemon.ExecContainer() error: %v", err)
		s.Write([]byte(fmt.Sprintf("%s not found\n", container)))
		s.Exit(255)
		return
	}
	go func() {
		for win := range winCh {
			log.Debugf("winCh: %v", win)
			sendControlResize(ws, win.Width, win.Height)
		}
	}()

	err = op.Wait()
	if err != nil {
		log.Errorf("op.Wait() error: %v", err)
		return
	}

	<-execArgs.DataDone
	opAPI := op.Get()

	ret := int(opAPI.Metadata["return"].(float64))
	log.Debugf("opAPI.Metadata[return]: %v", ret)
	//log.Debugf("opAPI.Metadata: %#v", opAPI)

	s.Exit(ret)
}

func sendControlResize(ws *websocket.Conn, width int, height int) {
	msg := api.InstanceExecControl{}
	msg.Command = "window-resize"
	msg.Args = make(map[string]string)
	msg.Args["width"] = strconv.Itoa(width)
	msg.Args["height"] = strconv.Itoa(height)

	ws.WriteJSON(msg)
}
