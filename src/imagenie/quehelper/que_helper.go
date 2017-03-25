package quehelper

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
)

type QueHelper struct {
	pgxpool *pgx.ConnPool
	qc      *que.Client
	workers *que.WorkerPool
}

func (self *QueHelper) Init(host, user, db_name, db_password string, number_of_workers int) error {
	config := pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     host,
			User:     user,
			Database: db_name,
			Password: db_password,
		},
		AfterConnect: que.PrepareStatements,
	}

	var err error
	self.pgxpool, err = pgx.NewConnPool(config)
	if err != nil {
		log.Error("Error in initializing connection pool for postgres", err)
		return err
	}

	self.qc = que.NewClient(self.pgxpool)
	wm := que.WorkMap{
		"PrintName": self.PrintJobProcessor,
	}
	self.workers = que.NewWorkerPool(self.qc, wm, number_of_workers)
	go self.workers.Start()
	return nil
}

func (self *QueHelper) Shutdown() {
	self.workers.Shutdown()
	self.pgxpool.Close()
}

type printNameArgs struct {
	Name string
}

func (self *QueHelper) PrintJobProcessor(j *que.Job) error {
	var args printNameArgs
	if err := json.Unmarshal(j.Args, &args); err != nil {
		return err
	}
	fmt.Printf("Hello %s!\n", args.Name)
	return nil
}

func (self *QueHelper) JobIt() error {
	args, err := json.Marshal(printNameArgs{Name: "bgentry"})
	if err != nil {
		return err
	}

	j := &que.Job{
		Type: "PrintName",
		Args: args,
	}

	if err := self.qc.Enqueue(j); err != nil {
		return err
	}
	return nil
}
