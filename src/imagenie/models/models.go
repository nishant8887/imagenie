package models

import (
	"github.com/jinzhu/gorm"
	"time"
)

// QueJobs Table
// priority    | smallint                 | not null default 100                                      | plain    |              |
// run_at      | timestamp with time zone | not null default now()                                    | plain    |              |
// job_id      | bigint                   | not null default nextval('que_jobs_job_id_seq'::regclass) | plain    |              |
// job_class   | text                     | not null                                                  | extended |              |
// args        | json                     | not null default '[]'::json                               | extended |              |
// error_count | integer                  | not null default 0                                        | plain    |              |
// last_error  | text                     |                                                           | extended |              |
// queue       | text                     | not null default ''::text                                 | extended |              |
// Indexes:
//     "que_jobs_pkey" PRIMARY KEY, btree (queue, priority, run_at, job_id)

type QueJobs struct {
	Priority   int32     `gorm:"primary_key;not null;default:100;" sql:"type:int"`
	RunAt      time.Time `gorm:"primary_key;not null;default:now()"`
	JobId      int64     `gorm:"primary_key;AUTO_INCREMENT"`
	JobClass   string    `gorm:"not null"`
	Args       string    `gorm:"not null;default:'[]'"`
	ErrorCount int32     `gorm:"not null;default:0"`
	Queue      string    `gorm:"primary_key;not null;default:''"`
	LastError  string
}

type User struct {
	gorm.Model
	UserName  string `gorm:"not null;unique_index;size:255"`
	FirstName string `gorm:"size:255"`
	LastName  string `gorm:"size:255"`
	Email     string `gorm:"not null;unique_index;size:255"`
	Password  string `gorm:"not null"`

	Images []Image
}

type Image struct {
	gorm.Model
	UserID    uint
	FileId    string `gorm:"not null;unique_index;size:40"`
	Extension string `gorm:"not null;size:5;default:'.jpg'"`
	Done      bool   `gorm:"not null;default:false`
}

func MigrateAll(db *gorm.DB) {
	db.AutoMigrate(&QueJobs{}, &User{}, &Image{})
}
