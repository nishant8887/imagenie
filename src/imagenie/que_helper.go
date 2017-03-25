package main

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/bgentry/que-go"
	"github.com/disintegration/imaging"
	"github.com/jackc/pgx"
	"imagenie/utils"
	"os"
	"path"
)

const (
	JOB_RETRIES = 3

	FOLDER_ORIGINAL = "original"
	FOLDER_RESIZED  = "resized"
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
		"ProcessImage": self.ProcessImage,
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

type ImageArgs struct {
	ImageId   string
	Extension string
	Path      string
}

func (self *QueHelper) UploadImage(image_path string, folder string) error {
	// Check for already uploaded image
	// Add some retries
	err := self.listener.awsHelper.UploadFile(image_path, folder)
	if err != nil {
		log.Error("Error in uploading image: ", err)
	}
	return err
}

func (self *QueHelper) ResizeImage(file_name, resized_file_name string) error {
	if _, err := os.Stat(resized_file_name); os.IsNotExist(err) {

		image, err := imaging.Open(file_name)
		if err != nil {
			log.Error("Error in opening file: ", file_name, err)
			return err
		}

		width := image.Bounds().Dx()
		height := image.Bounds().Dy()
		log.Info("Width of the image: ", width)
		log.Info("Height of the image: ", height)

		if width > height {
			image = imaging.Resize(image, 640, 0, imaging.Lanczos)
		} else {
			image = imaging.Resize(image, 0, 480, imaging.Lanczos)
		}

		err = imaging.Save(image, resized_file_name)
		if err != nil {
			log.Error("Error in saving image: ", resized_file_name)
			return err
		}
	}
	return nil
}

func (self *QueHelper) ProcessImage(j *que.Job) error {
	var args ImageArgs
	if err := json.Unmarshal(j.Args, &args); err != nil {
		log.Error("Error in parsing job arguments", err)
		return nil
	}

	file_name := args.Path
	resized_file_name := path.Clean(fmt.Sprintf("%s/%s-resized%s", self.listener.settings.TmpPath, args.ImageId, args.Extension))

	// Check for limited number of retries
	if j.ErrorCount >= JOB_RETRIES {

		// Add code to delete uploaded files if any
		self.listener.awsHelper.DeleteFile(file_name, FOLDER_ORIGINAL)
		self.listener.awsHelper.DeleteFile(resized_file_name, FOLDER_RESIZED)

		utils.DeleteFile(file_name)
		utils.DeleteFile(resized_file_name)

		log.Error("Cannot process image: ", args.ImageId)
		return nil
	}

	x := make(chan bool, 2)
	go func() {
		err := self.UploadImage(file_name, FOLDER_ORIGINAL)
		if err != nil {
			x <- false
			return
		}
		x <- true
	}()

	go func() {
		err := self.ResizeImage(file_name, resized_file_name)
		if err != nil {
			x <- false
			return
		}

		err = self.UploadImage(resized_file_name, FOLDER_RESIZED)
		if err != nil {
			x <- false
			return
		}
		x <- true
	}()

	x1 := <-x
	x2 := <-x

	if !x1 || !x2 {
		return errors.New("error_in_processing_image")
	}

	// Mark the image as done
	err := self.listener.db.Table("images").Where("file_id = ?", args.ImageId).Update("done", true).Error
	if err != nil {
		log.Error("Error in updating status of image: ", err)
		return errors.New("error_in_updating_image_status")
	}

	utils.DeleteFile(file_name)
	utils.DeleteFile(resized_file_name)
	return nil
}

func (self *QueHelper) DoProcess(data ImageArgs) error {
	args, err := json.Marshal(data)
	if err != nil {
		log.Error("Error marshalling json: ", err)
		return err
	}

	j := &que.Job{
		Type: "ProcessImage",
		Args: args,
	}

	// Add more retries
	if err := self.qc.Enqueue(j); err != nil {
		return err
	}
	return nil
}
