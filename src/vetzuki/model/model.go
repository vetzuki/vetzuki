package model

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/xid"
	"github.com/vetzuki/vetzuki/ldap"

	"log"
	"os"
	"time"
)

const (
	envDBHost     = "DB_HOST"
	envDBPort     = "DB_PORT"
	envDBUsername = "DB_USERNAME"
	envDBPassword = "DB_PASSWORD"
	envDBName     = "DB_NAME"
	envSSLMode    = "DB_SSL_MODE"
)

var (
	connection            *sql.DB
	dbName                = "vetzuki"
	connectionTemplate    = "postgres://%s:%s@%s:%s/%s"
	sslConnectionTemplate = "postgres://%s:%s@%s:%s/%s?sslmode=%s"
	dbHost                = "localhost"
	dbPort                = "5432"
	dbUser                = "admin"
	dbPassword            = "admin"
	sslMode               = ""
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	Connect()
}

// Connect : Create a new database connection
func Connect() {
	var connectionString string
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("error: failed to locate .env")
	}

	dbHost = os.Getenv(envDBHost)
	dbPort = os.Getenv(envDBPort)
	dbUser = os.Getenv(envDBUsername)
	dbName = os.Getenv(envDBName)
	dbPassword = os.Getenv(envDBPassword)
	sslMode = os.Getenv(envSSLMode)
	if len(sslMode) == 0 {
		connectionString = fmt.Sprintf(connectionTemplate, dbUser, dbPassword, dbHost, dbPort, dbName)
	} else {
		connectionString = fmt.Sprintf(sslConnectionTemplate, dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)
	}
	log.Printf("debug: connecting to %s", connectionString)
	if c, err := sql.Open("postgres", connectionString); err != nil {
		log.Fatalf("fatal: unable to connect to %s: %s", connectionString, err)
	} else {
		connection = c
	}
}

// CreateEmployerProspect : Create an EmployerExam, Prospect, and LDAP user
func CreateEmployerProspect(employerID, examID int64, name, email string) (*EmployerProspect, bool) {
	employer, ok := findEmployerByID(employerID)
	if !ok {
		log.Printf("error: failed to locate employer %d", employerID)
		return nil, false
	}
	exam, ok := findExamByID(examID)
	if !ok {
		log.Printf("error: failed to find exam %d", examID)
		return nil, false
	}
	employerExam, ok := createEmployerExam(employer, exam)
	if !ok {
		log.Printf("error: failed to create employerExam for employer %s on exam %s", employer.Name, exam.Name)
		return nil, false
	}
	prospect, ok := createProspect(employerExam, name, email)
	if !ok {
		log.Printf("error: failed to create prospect %s", email)
	}
	c := ldap.Connect()
	if c == nil {
		log.Printf("error: failed to create LDAP connection")
		return nil, false
	}
	user, ok := ldap.CreateProspect(c, prospect.URL)
	if !ok {
		log.Printf("error: failed to create ldap user %s for Prospect %s", prospect.URL, prospect.Email)
		return nil, false
	}
	log.Printf("debug: created LDAP user %s", user.DN)
	return createEmployerProspect(employerExam, prospect)
}

func createEmployerProspect(employerExam *EmployerExam, prospect *Prospect) (*EmployerProspect, bool) {
	ep := &EmployerProspect{
		ProspectID:     prospect.ID,
		EmployerID:     employerExam.EmployerID,
		EmployerExamID: employerExam.ID,
	}
	row := connection.QueryRow(`
	  INSERT into employer_prospect (prospect_id, employer_id, employer_exam_id)
	  VALUES ($1, $2, $3)`,
		ep.ProspectID,
		ep.EmployerID,
		ep.EmployerExamID,
	)

	if row == nil {
		log.Printf("error: failed to create employerProspect %d,%d,%d: %v",
			ep.ProspectID, ep.EmployerID, ep.EmployerExamID, row)
		return nil, false
	}
	return ep, true
}
func createProspect(employerExam *EmployerExam, name, email string) (*Prospect, bool) {
	prospect := &Prospect{
		Name:           name,
		Email:          email,
		URL:            xid.New().String(),
		EmployerID:     employerExam.EmployerID,
		EmployerExamID: employerExam.ExamID,
	}
	err := connection.QueryRow(`
	  INSERT into prospect (name, email, url, employer_id, employer_exam_id)
	  VALUES ($1, $2, $3, $4, $5)
	  RETURNING id, created, modified`,
		prospect.Name,
		prospect.Email,
		prospect.URL,
		employerExam.EmployerID,
		employerExam.ExamID,
	).Scan(&prospect.ID, &prospect.Created, &prospect.Modified)
	if err != nil {
		log.Printf("error: unable to insert prospect %s for employerExam %d: %s", email, employerExam.ID, err)
		return nil, false
	}
	return prospect, true
}
func createEmployerExam(employer *Employer, exam *Exam) (*EmployerExam, bool) {
	employerExam := &EmployerExam{
		EmployerID: employer.ID,
		ExamID:     exam.ID,
	}
	err := connection.QueryRow(`
	  INSERT into employer_exam (employer_id, exam_id)
	  VALUES ($1, $2)
	  RETURNING id, created, modified`, employer.ID, exam.ID,
	).Scan(&employerExam.ID, &employerExam.Created, &employerExam.Modified)
	if err != nil {
		log.Printf("error: while creating employerExam for %s on exam %s: %s", employer.Name, exam.Name, err)
		return employerExam, false
	}
	return employerExam, true
}
func findProspectByID(id int64) (*Prospect, bool) {
	prospect := &Prospect{ID: id}
	err := connection.QueryRow(`
	  SELECT name, email, url, employer_id, employer_exam_id, created, modified
	  FROM prospect
	  WHERE id = $1
	`, id).Scan(
		&prospect.Name,
		&prospect.Email,
		&prospect.URL,
		&prospect.EmployerID,
		&prospect.EmployerExamID,
		&prospect.Created,
		&prospect.Modified,
	)
	if err != nil {
		log.Printf("error: failed to locate prospect %d", id)
		return nil, false
	}
	return prospect, true
}
func findExamByID(id int64) (*Exam, bool) {
	exam := &Exam{ID: id}
	err := connection.QueryRow(`
	  SELECT name, description, created, modified
	  FROM exam
	  WHERE id = $1`, id).Scan(
		&exam.Name,
		&exam.Description,
		&exam.Created,
		&exam.Modified,
	)
	if err != nil {
		log.Printf("error: failed to locate exam %d: %s", id, err)
		return nil, false
	}
	return exam, true
}
func findEmployerByID(id int64) (*Employer, bool) {
	employer := &Employer{ID: id}
	err := connection.QueryRow(`
	  SELECT name,email,  billing_email, billing_state, created, modified
	  FROM employer where id = $1`, id).Scan(
		&employer.Name,
		&employer.Email,
		&employer.BillingEmail,
		&employer.BillingState,
		&employer.Created,
		&employer.Modified,
	)
	if err != nil {
		log.Printf("error: unable to find employer %d: %s", id, err)
		return nil, false
	}
	return employer, true
}
func findEmployer(email string) (*Employer, bool) {
	employer := &Employer{Email: email}
	row := connection.QueryRow(`
	  SELECT id,name,billing_email, billing_state, created, modified
	  FROM employer
	  WHERE email = $1`,
		email,
	)
	if row == nil {
		log.Printf("error: no such employer %s", email)
		return employer, false
	}
	err := row.Scan(
		&employer.ID,
		&employer.Name,
		&employer.BillingEmail,
		&employer.BillingState,
		&employer.Created,
		&employer.Modified,
	)

	if err != nil {
		log.Printf("error: while finding employer by %s: %s", email, err)
		return employer, false
	}

	return employer, true
}

// Employer : An employer can hire prospects
type Employer struct {
	ID           int64     `sql:"id" json:"id"`
	Created      time.Time `sql:"created" json:"created"`
	Modified     time.Time `sql:"modified" json:"modified"`
	Name         string    `sql:"name" json:"name"`
	Email        string    `sql:"email" json:"email"`
	BillingEmail string    `sql:"billing_email" json:"billingEmail"`
	BillingID    int64     `sql:"billing_id" json:"billingID"`
	BillingState int       `sql:"billing_state" json:"billingState"`
}

// Exam : An exam describes a type of exam for a prospect
type Exam struct {
	ID          int64     `sql:"id" json:"id"`
	Created     time.Time `sql:"created" json:"created"`
	Modified    time.Time `sql:"modified" json:"modified"`
	Name        string    `sql:"name" json:"name"`
	Description string    `sql:"description" json:"description"`
}

// EmployerExam : An exam issued by an employer
type EmployerExam struct {
	ID         int64     `sql:"id" json:"id"`
	Created    time.Time `sql:"created" json:"created"`
	Modified   time.Time `sql:"modified" json:"modified"`
	EmployerID int64     `sql:"employer_id" json:"employerID"`
	ExamID     int64     `sql:"exam_id" json:"examID"`
}

// Prospect : A prospective hire to an employer
type Prospect struct {
	ID             int64     `sql:"id" json:"id"`
	Created        time.Time `sql:"created" json:"created"`
	Modified       time.Time `sql:"modified" json:"modified"`
	Name           string    `sql:"name" json:"name"`
	Email          string    `sql:"email" json:"email"`
	URL            string    `sql:"url" json:"url"`
	EmployerID     int64     `sql:"employer_id" json:"employerID"`
	EmployerExamID int64     `sql:"employer_exam_id" json:"employerExamID"`
}

// EmployerProspect : A prospect of an employer
type EmployerProspect struct {
	ProspectID     int64 `sql:"prospect_id" json:"prospectID"`
	EmployerID     int64 `sql:"employer_id" json:"employerID"`
	EmployerExamID int64 `sql:"employer_exam_id" json:"employerExamID"`
}