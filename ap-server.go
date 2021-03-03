package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
)

const (
	VERSION = "0.1"
)
const (
	MONGO = iota
	FILES
)
const DEFAULT_DB = MONGO
const DEFAULT_CONNECTION_STRING = ""

type Server struct {
	ctx      context.Context
	wg       sync.WaitGroup
	
	hostname string
	opts struct {
		dbg            int
		me             string
		//--
		http           string
		db             string
		auto           bool
		fix            bool
		typeDB int
		mongo string
	}

	api     *Api
	db      idb
}

func NewIDB(server * Server) (idb, error){
	if server.opts.typeDB == MONGO{
		return NewMongoDB(server),nil
	}
	if server.opts.typeDB == FILES{
		return NewDB(server),nil
	}
	return nil, errors.New(fmt.Sprint("Forbidden DB type. Got ",server.opts.typeDB, " expected ",MONGO,"or ",FILES))
}

func main() {
	var err error
	S := &Server{}

	S.ctx = context.Background()
	S.hostname, err = os.Hostname()
	if err != nil { dieErr("main", err) }

	// command-line args
	flag.IntVar(&S.opts.dbg, "dbg", 2, "debugging level")
	flag.StringVar(&S.opts.me, "me", S.hostname, "my identity, e.g. name of this host")
	flag.StringVar(&S.opts.http, "http", ":30000", "listen on given HTTP endpoint")
	flag.StringVar(&S.opts.db, "db", "localhost", "path to filesystem database or IP address of the database")
	flag.BoolVar(&S.opts.auto, "auto", true, "automatically add first seen MAC on a port")
	flag.BoolVar(&S.opts.fix, "fix", true, "fix missing keys in profiles (use old values)")
	flag.IntVar(&S.opts.typeDB, "db-type",DEFAULT_DB,fmt.Sprint("Which database is used. Mongo = ",MONGO," local file storage : ",FILES, "."))
	flag.Parse()
	dbgSet(S.opts.dbg)
	S.db, err = NewIDB(S)
	if err !=nil{
		dieErr("INIT", err)
	}
	S.api = NewApi(S)
	if len(S.opts.http) > 0 {
		S.wg.Add(1)
		go S.api.ServeHttp(S.opts.http)
	}

	S.wg.Wait()
}
