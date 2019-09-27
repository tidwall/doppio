package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/dgraph-io/ristretto"
	"github.com/tidwall/redcon"
	"github.com/tidwall/redlog"
)

var log = redlog.New(os.Stderr)
var cache *ristretto.Cache

func main() {
	var port int
	var capacity string

	flag.IntVar(&port, "port", 6380, "Server port")
	flag.StringVar(&capacity, "s", "1gb", "Cache capacity of the database, such as 4gb, 500mb, etc.")
	flag.Parse()
	x, err := humanize.ParseBytes(capacity)
	if err != nil {
		log.Fatalf("Invalid cache capacity %v", capacity)
	}

	cache, err = ristretto.NewCache(&ristretto.Config{
		MaxCost:     int64(x),
		NumCounters: int64(x) * 10,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		fmt.Printf("\n%s\n\n", strings.Join([]string{
			"d8888b.  .d88b.  d8888b. d8888b. d888888b  .d88b.  ",
			"88  `8D .8P  Y8. 88  `8D 88  `8D   `88'   .8P  Y8. ",
			"88   88 88    88 88oodD' 88oodD'    88    88    88 ",
			"88  .8D `8b  d8' 88      88        .88.   `8b  d8' ",
			"Y8888D'  `Y88P'  88      88      Y888888P  `Y88P'  ",
		}, "\n"))
		log.Printf("Server started on port %d (%s/%s, %d threads, %s capacity)\n",
			port, runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), humanize.Bytes(x))
	}()
	log.Fatal(redcon.ListenAndServe(fmt.Sprintf(":%d", port),
		func(conn redcon.Conn, cmd redcon.Command) {
			handleCommand(conn, cmd)
		}, nil, nil))
}

func handleCommand(conn redcon.Conn, cmd redcon.Command) {
	switch strings.ToUpper(string(cmd.Args[0])) {
	default:
		conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
	case "QUIT":
		conn.WriteString("OK")
		conn.Close()
	case "SHUTDOWN":
		conn.WriteString("OK")
		log.Fatal("Shutting server down, bye bye")
	case "PING":
		if len(cmd.Args) == 1 {
			conn.WriteString("PONG")
		} else if len(cmd.Args) == 2 {
			conn.WriteBulk(cmd.Args[1])
		} else {
			conn.WriteError("ERR invalid number of arguments")
		}
	case "ECHO":
		if len(cmd.Args) != 2 {
			conn.WriteError("ERR invalid number of arguments")
		} else if len(cmd.Args) == 2 {
			conn.WriteBulk(cmd.Args[1])
		}
	case "SET":
		if len(cmd.Args) != 3 {
			conn.WriteError("ERR invalid number of arguments")
		} else {
			cache.Set(string(cmd.Args[1]), string(cmd.Args[2]), int64(len(cmd.Args[2])))
			conn.WriteString("OK")
		}
	case "GET":
		if len(cmd.Args) != 2 {
			conn.WriteError("ERR invalid number of arguments")
		} else if val, ok := cache.Get(string(cmd.Args[1])); !ok {
			conn.WriteNull()
		} else {
			conn.WriteBulkString(val.(string))
		}
	case "DEL":
		if len(cmd.Args) < 2 {
			conn.WriteError("ERR invalid number of arguments")
		} else {
			for i := 1; i < len(cmd.Args); i++ {
				cache.Del(string(cmd.Args[i]))
			}
			conn.WriteInt(len(cmd.Args) - 1)
		}
	}
}
