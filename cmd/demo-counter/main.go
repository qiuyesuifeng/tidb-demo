package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/siddontang/go/sync2"
	"github.com/unrolled/render"
	"github.com/urfave/negroni"
)

// NewConfig creates a new config.
func NewConfig() *Config {
	cfg := &Config{}
	cfg.FlagSet = flag.NewFlagSet("counter", flag.ContinueOnError)
	fs := cfg.FlagSet

	fs.StringVar(&cfg.configFile, "config", "", "Config file")

	return cfg
}

// DBConfig is the DB configuration.
type DBConfig struct {
	Host string `toml:"host" json:"host"`

	User string `toml:"user" json:"user"`

	Password string `toml:"password" json:"password"`

	Port int `toml:"port" json:"port"`

	Name string `toml:"name" json:"name"`
}

// Config is the configuration.
type Config struct {
	*flag.FlagSet `json:"-"`

	Addr string `toml:"addr" json:"addr"`

	Tps int64 `toml:"tps" json:"tps"`

	DB DBConfig `toml:"db" json:"db"`

	configFile string
}

func (c *Config) Parse(arguments []string) error {
	// Parse first to get config file.
	err := c.FlagSet.Parse(arguments)
	if err != nil {
		return errors.Trace(err)
	}

	// Load config file if specified.
	if c.configFile != "" {
		err = c.configFromFile(c.configFile)
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Parse again to replace with command line options.
	err = c.FlagSet.Parse(arguments)
	if err != nil {
		return errors.Trace(err)
	}

	if len(c.FlagSet.Args()) != 0 {
		return errors.Errorf("'%s' is an invalid flag", c.FlagSet.Arg(0))
	}

	return nil
}

// configFromFile loads config from file.
func (c *Config) configFromFile(path string) error {
	_, err := toml.DecodeFile(path, c)
	return errors.Trace(err)
}

type counter struct {
	sync.Mutex

	cfg *Config

	wg sync.WaitGroup

	db *sql.DB

	done chan struct{}
	quit chan struct{}

	closed sync2.AtomicBool
}

func newCounter(cfg *Config) *counter {
	counter := new(counter)
	counter.cfg = cfg
	counter.closed.Set(false)
	counter.done = make(chan struct{})
	counter.quit = make(chan struct{})
	return counter
}

func (c *counter) start() error {
	var err error
	c.db, err = createDB(c.cfg.DB)
	if err != nil {
		return errors.Trace(err)
	}

	err = c.init()
	if err != nil {
		return errors.Trace(err)
	}

	c.wg.Add(1)

	go c.run()

	return nil
}

func (c *counter) run() {
	defer c.wg.Done()

	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-c.quit:
			c.done <- struct{}{}
			return
		case <-timer.C:
			c.doJob(c.cfg.Tps)
		}
	}
}

func (c *counter) init() error {
	sql := "drop table if exists counter;"
	err := execSQL(c.db, sql)
	if err != nil {
		return errors.Trace(err)
	}

	sql = "create table counter(id int primary key, value bigint);"
	err = execSQL(c.db, sql)
	if err != nil {
		return errors.Trace(err)
	}

	sql = "insert counter (id,value) values (1,0);"
	err = execSQL(c.db, sql)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *counter) doJob(count int64) {
	number := rand.Int63n(count)

	for i := 0; i < int(number); i++ {
		sql := "update counter set value = value + 1 where id = 1;"
		err := execSQL(c.db, sql)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (c *counter) isClosed() bool {
	return c.closed.Get()
}

func (c *counter) close() {
	c.Lock()
	defer c.Unlock()

	if c.isClosed() {
		return
	}

	close(c.quit)

	<-c.done

	c.wg.Wait()

	closeDB(c.db)
}

func execSQL(db *sql.DB, sql string) error {
	if len(sql) == 0 {
		return nil
	}

	_, err := db.Exec(sql)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func getCount(db *sql.DB, query string) (int64, error) {
	rows, err := db.Query(query)
	if err != nil {
		return -1, errors.Trace(err)
	}

	var count int64
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return -1, errors.Trace(err)
		}
	}

	if rows.Err() != nil {
		return -1, errors.Trace(rows.Err())
	}

	return count, nil
}

func createDB(cfg DBConfig) (*sql.DB, error) {
	dbDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	db, err := sql.Open("mysql", dbDSN)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return db, nil
}

func closeDB(db *sql.DB) error {
	return errors.Trace(db.Close())
}

type Count struct {
	Count int64 `json:"count"`
}

type counterHandler struct {
	rd *render.Render
	db *sql.DB
}

func newCounterHandler(rd *render.Render, db *sql.DB) *counterHandler {
	return &counterHandler{
		rd: rd,
		db: db,
	}
}

func (h *counterHandler) Get(w http.ResponseWriter, r *http.Request) {
	sql := "select value from counter where id = 1;"
	value, _ := getCount(h.db, sql)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	h.rd.JSON(w, http.StatusOK, &Count{Count: value})
}

func startHTTP(c *counter) {
	rd := render.New(render.Options{
		IndentJSON: true,
	})

	engine := negroni.New()

	recovery := negroni.NewRecovery()
	engine.Use(recovery)

	router := mux.NewRouter()
	counterHandler := newCounterHandler(rd, c.db)
	router.HandleFunc("/api/v1/counter", counterHandler.Get).Methods("GET")
	engine.UseHandler(router)

	http.ListenAndServe(c.cfg.Addr, engine)
}

func main() {
	cfg := NewConfig()
	err := cfg.Parse(os.Args[1:])
	switch errors.Cause(err) {
	case nil:
	case flag.ErrHelp:
		os.Exit(0)
	default:
		log.Errorf("parse cmd flags err %s\n", err)
		os.Exit(2)
	}

	log.Infof("%v", cfg)

	counter := newCounter(cfg)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		sig := <-sc
		log.Infof("Got signal [%d] to exit.", sig)
		counter.close()
		os.Exit(2)
	}()

	err = counter.start()
	if err != nil {
		log.Fatal(errors.ErrorStack(err))
	}

	startHTTP(counter)
}
