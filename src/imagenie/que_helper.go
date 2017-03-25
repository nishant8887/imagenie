package main

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
)

type QueHelper struct {
	listener *ImagenieListener
	pgxpool  *pgx.ConnPool
	qc       *que.Client
	workers  *que.WorkerPool
}

func (self *QueHelper) Init(listener *ImagenieListener) error {
	self.listener = listener
	config := pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     listener.settings.DbHost,
			User:     listener.settings.DbUser,
			Database: listener.settings.DbName,
			Password: listener.settings.DbPassword,
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
		"ResizeImage": self.ResizeImage,
	}
	self.workers = que.NewWorkerPool(self.qc, wm, listener.settings.Workers)
	go self.workers.Start()
	return nil
}

func (self *QueHelper) Shutdown() {
	self.workers.Shutdown()
	self.pgxpool.Close()
	self.listener = nil
}

type ResizeImageArgs struct {
	ImageId string
	Path    string
}

func (self *QueHelper) ResizeImage(j *que.Job) error {
	// Check for limited number of retries

	var args ResizeImageArgs
	if err := json.Unmarshal(j.Args, &args); err != nil {
		return err
	}
	return nil
}

func (self *QueHelper) DoResize(data ResizeImageArgs) error {
	args, err := json.Marshal(data)
	if err != nil {
		log.Error("Error marshalling json: ", err)
		return err
	}

	j := &que.Job{
		Type: "ResizeImage",
		Args: args,
	}

	// Add more retries
	if err := self.qc.Enqueue(j); err != nil {
		return err
	}
	return nil
}
