package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/dgraph-io/ristretto"
	"github.com/dustin/go-humanize"
	"github.com/tidwall/evio"
	"github.com/tidwall/redcon"
	"github.com/tidwall/redlog"
)

var log = redlog.New(os.Stderr)
var cache *ristretto.Cache
var port int
var capacity uint64
var threads int

func main() {
	var capflag string
	var single bool
	flag.IntVar(&port, "p", 6380, "Server port")
	flag.BoolVar(&single, "single-threaded", runtime.GOMAXPROCS(0) == 1,
		"Run in Single-threaded mode")
	flag.StringVar(&capflag, "s", "1gb",
		"Cache capacity of the database, such as 4gb, 500mb, etc.")
	flag.Parse()
	x, err := humanize.ParseBytes(capflag)
	if err != nil {
		log.Fatalf("Invalid cache capacity %v", capflag)
	}
	capacity = uint64(x)
	cache, err = ristretto.NewCache(&ristretto.Config{
		MaxCost:     int64(x),
		NumCounters: int64(x) * 10,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}
	if single {
		threads = 1
	} else {
		threads = runtime.GOMAXPROCS(0)
	}

	if threads == 1 {
		useEvio()
	} else {
		useRedcon()
	}
}

func printMast() {
	fmt.Printf("\n%s\n\n", strings.Join([]string{
		"d8888b.  .d88b.  d8888b. d8888b. d888888b  .d88b.  ",
		"88  `8D .8P  Y8. 88  `8D 88  `8D   `88'   .8P  Y8. ",
		"88   88 88    88 88oodD' 88oodD'    88    88    88 ",
		"88  .8D `8b  d8' 88      88        .88.   `8b  d8' ",
		"Y8888D'  `Y88P'  88      88      Y888888P  `Y88P'  ",
	}, "\n"))

	threadss := fmt.Sprintf("%d threads", threads)
	if threads == 1 {
		threadss = "single-threaded"
	}
	log.Printf("Server started on port %d (%s/%s, %s, %s capacity)\n",
		port, runtime.GOOS, runtime.GOARCH, threadss,
		humanize.Bytes(capacity))
}

func useEvio() {
	var events evio.Events
	events.NumLoops = threads

	events.Serving = func(srv evio.Server) (action evio.Action) {
		printMast()
		return
	}

	events.Opened = func(ec evio.Conn) (
		out []byte, opts evio.Options, action evio.Action,
	) {
		ec.SetContext(&client{})
		return
	}

	events.Closed = func(ec evio.Conn, err error) (action evio.Action) {
		return
	}

	events.Data = func(ec evio.Conn, in []byte) (
		out []byte, action evio.Action,
	) {
		c := ec.Context().(*client)
		data := c.is.Begin(in)
		var complete bool
		var err error
		var args [][]byte
		for action == evio.None {
			complete, args, _, data, err =
				redcon.ReadNextCommand(data, args[:0])
			if err != nil {
				action = evio.Close
				out = redcon.AppendError(out, err.Error())
				break
			}
			if !complete {
				break
			}
			if len(args) > 0 {
				out, action = handleCommand(out, args)
			}
		}
		c.is.End(data)
		return
	}
	log.Fatal(evio.Serve(events, fmt.Sprintf("tcp://:%d", port)))
}

type client struct {
	is   evio.InputStream
	addr string
}

func useRedcon() {
	go func() {
		printMast()
	}()
	log.Fatal(redcon.ListenAndServe(fmt.Sprintf(":%d", port),
		func(conn redcon.Conn, cmd redcon.Command) {
			out, action := handleCommand(nil, cmd.Args)
			if len(out) > 0 {
				conn.WriteRaw(out)
			}
			if action == evio.Close {
				conn.Close()
			}
		}, nil, nil))
}

func handleCommand(out []byte, args [][]byte) ([]byte, evio.Action) {
	var action evio.Action
	switch strings.ToUpper(string(args[0])) {
	default:
		out = redcon.AppendError(out,
			"ERR unknown command '"+string(args[0])+"'")
	case "QUIT":
		out = redcon.AppendOK(out)
		action = evio.Close
	case "SHUTDOWN":
		out = redcon.AppendOK(out)
		log.Fatal("Shutting server down, bye bye")
	case "PING":
		if len(args) == 1 {
			out = redcon.AppendString(out, "PONG")
		} else if len(args) == 2 {
			out = redcon.AppendBulk(out, args[1])
		} else {
			out = redcon.AppendError(out, "ERR invalid number of arguments")
		}
	case "ECHO":
		if len(args) != 2 {
			out = redcon.AppendError(out, "ERR invalid number of arguments")
		} else if len(args) == 2 {
			out = redcon.AppendBulk(out, args[1])
		}
	case "SET":
		if len(args) != 3 {
			out = redcon.AppendError(out, "ERR invalid number of arguments")
		} else {
			cache.Set(string(args[1]), string(args[2]), int64(len(args[2])))
			out = redcon.AppendOK(out)
		}
	case "GET":
		if len(args) != 2 {
			out = redcon.AppendError(out, "ERR invalid number of arguments")
		} else if val, ok := cache.Get(string(args[1])); !ok {
			out = redcon.AppendNull(out)
		} else {
			out = redcon.AppendBulkString(out, val.(string))
		}
	case "DEL":
		if len(args) < 2 {
			out = redcon.AppendError(out, "ERR invalid number of arguments")
		} else {
			for i := 1; i < len(args); i++ {
				cache.Del(string(args[i]))
			}
			out = redcon.AppendInt(out, int64(len(args)-1))
		}
	}
	return out, action
}
