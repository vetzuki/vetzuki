package model

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Side effects only
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
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("warning: failed to locate .env")
	}

	dbHost = os.Getenv(envDBHost)
	dbPort = os.Getenv(envDBPort)
	dbUser = os.Getenv(envDBUsername)
	dbName = os.Getenv(envDBName)
	dbPassword = os.Getenv(envDBPassword)
	sslMode = os.Getenv(envSSLMode)
	if len(sslMode) == 0 {
		sslMode = "disable"
	}
	connectionString = fmt.Sprintf(sslConnectionTemplate, dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

	log.Printf("debug: connecting to %s", connectionString)
	if c, err := sql.Open("postgres", connectionString); err != nil {
		log.Fatalf("fatal: unable to connect to %s: %s", connectionString, err)
	} else {
		connection = c
	}
}

// CreateOrganization : Create an organization
func CreateOrganization(name string) (*Organization, bool) {
	log.Printf("debug: creating organization %s", name)
	organization := &Organization{
		Name: name,
	}
	err := connection.QueryRow(`
	INSERT INTO organization
	(name)
	VALUES ($1)
	RETURNING id, created, modified`, name,
	).Scan(&organization.ID, &organization.Created, &organization.Modified)

	if err != nil {
		log.Printf("error: failed to create organization %s: %s", name, err)
		return nil, false
	}
	return organization, true
}

// CreateOrganizationalRole : Create a organizational role
func CreateOrganizationalRole(employer *Employer, organization *Organization, role int) (*OrganizationalRole, bool) {
	log.Printf("debug: creating the organizational role %d in %s for %s",
		role,
		organization.Name,
		employer.Email)
	orgRole := &OrganizationalRole{
		EmployerID:     employer.ID,
		OrganizationID: organization.ID,
		Role:           role,
	}
	err := connection.QueryRow(`
	INSERT INTO organizational_role
	(employer_id, organization_id, role)
	VALUES ($1, $2, $3)
	RETURNING created, modified`,
		employer.ID,
		organization.ID,
		role,
	).Scan(
		&orgRole.Created,
		&orgRole.Modified,
	)
	if err != nil {
		log.Printf("error: failed to create org role %d for %s in %s: %s",
			role,
			employer.Name,
			organization.Name,
			err)
		return nil, false
	}
	return orgRole, true
}

// CreateEmployer : Create an employer
func CreateEmployer(name, email string) (*Employer, bool) {
	log.Printf("debug: creating employer %s", email)
	employer := &Employer{
		Name:  name,
		Email: email,
	}
	err := connection.QueryRow(`
	INSERT INTO employer
	(name, email)
	VALUES ($1, $2)
	RETURNING id, created, modified`,
	).Scan(
		employer.ID,
		employer.Created,
		employer.Modified,
	)
	if err != nil {
		log.Printf("error: failed to create employer %s: %s", email, err)
		return nil, false
	}
	return employer, true
}

// FindEmployerByEmail : Find an employer by email
func FindEmployerByEmail(email string) ([]*Employer, bool) {
	log.Printf("debug: finding employer %s", email)
	rows, err := connection.Query(`
	SELECT id, name, email, organization_id, created, modified
	FROM employer
	WHERE email = $1`, email)
	if err != nil {
		log.Printf("error: failed to find employer %s: %s", email, err)
		return nil, false
	}
	defer rows.Close()
	// TODO: implement sql.Scanner : Employer.Scan(interface{}) error
	members := []*Employer{}
	for rows.Next() {
		e := &Employer{}
		err := rows.Scan(
			&e.ID,
			&e.Name,
			&e.Email,
			&e.OrganizationID,
			&e.Created,
			&e.Modified,
		)
		if err != nil {
			log.Printf("error: failed to scan to employer: %s", err)
		}
	}
	return members, true
}

// GetMembers : Get all employers of an organization
func (o *Organization) GetMembers() ([]*Employer, bool) {
	log.Printf("debug: getting members of %s", o.Name)
	rows, err := connection.Query(`
	SELECT id, name, email, organization_id, created, modified
	FROM organization
	WHERE organization_id = $1`)
	if err != nil {
		log.Printf("error: failed to find members of %s: %s", o.Name, err)
		return nil, false
	}
	defer rows.Close()
	members := []*Employer{}
	for rows.Next() {
		e := &Employer{}
		err := rows.Scan(
			&e.ID,
			&e.Name,
			&e.Email,
			&e.OrganizationID,
			&e.Created,
			&e.Modified,
		)
		if err != nil {
			log.Printf("error: failed to scan to employer: %s", err)
		}
	}
	return members, true
}

// FindOrganizationByName : Find an organization by name
func FindOrganizationByName(name string) ([]*Organization, bool) {
	log.Printf("debug: finding employers named %s", name)
	rows, err := connection.Query(`
	SELECT id, name, created, modified
	FROM organization
	WHERE name = $1`, name)
	if err != nil {
		log.Printf("error: failed to find organizations named %s: %s", name, err)
		return nil, false
	}
	defer rows.Close()

	organizations := []*Organization{}
	for rows.Next() {
		org := &Organization{}
		err := rows.Scan(
			&org.ID,
			&org.Name,
			&org.Created,
			&org.Modified,
		)
		if err != nil {
			log.Printf("error: unable to cast row data to organization: %s", err)
			return nil, false
		}
	}
	return organizations, true
}

// GetRole : Get the role of an employee in their organization
func (e *Employer) GetRole() (*OrganizationalRole, bool) {
	log.Printf("debug: finding organization for %s", e.Email)
	rows, err := connection.Query(`
	SELECT employer_id, organization_id, role, created, modified
	FROM organization_role
	WHERE employer_id = $1 and organization_id = $2`,
		e.ID,
		e.OrganizationID,
	)
	if err != nil {
		log.Printf("error: faild to find role for %s: %s", e.Email, err)
		return nil, false
	}
	orgRole := &OrganizationalRole{}
	counter := 1
	for rows.Next() {
		if counter == 1 {
			err := rows.Scan(
				&orgRole.EmployerID,
				&orgRole.OrganizationID,
				&orgRole.Role,
				&orgRole.Created,
				&orgRole.Modified,
			)
			if err != nil {
				log.Printf("error: unable to restore organizational role for %s: %s",
					e.Email, err)
			}
		}
		counter++
	}
	if counter > 1 {
		log.Printf("warning: more than one role was found for %s in orgID %d",
			e.Email,
			e.OrganizationID)
	}
	return orgRole, false
}

// GetOrganization : Get an organization by ID
func GetOrganization(id int) (*Organization, bool) {
	log.Printf("debug: finding organization %d", id)
	rows, err := connection.Query(`
	SELECT id, name, created, modified
	FROM organization
	WHERE id = $1`, id)
	if err != nil {
		log.Printf("error: failed to find organization %d: %s", id, err)
		return nil, false
	}
	counter := 1
	org := &Organization{}
	for rows.Next() {
		if counter == 1 {
			err := rows.Scan(
				&org.ID,
				&org.Name,
				&org.Created,
				&org.Modified,
			)
			if err != nil {
				log.Printf("error: failed to scan organization %d: %s", id, err)
				return nil, false
			}
		}
		counter++
	}
	if counter > 1 {
		log.Printf("warning: found %d matches for organization %d", counter, id)
	}
	return org, true
}

// CreateEmployerProspect : Create an EmployerExam, Prospect, and LDAP user
func CreateEmployerProspect(employerID, examID int64, name, email, role string) (*EmployerProspect, bool) {
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
	prospect, ok := createProspect(employerExam, name, email, role)
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
	ep, ok := createEmployerProspect(employerExam, prospect)
	ep.Prospect = prospect
	ep.Employer = employer
	return ep, ok
}

// GetProspect : Get a prospect by their URL
func GetProspect(prospectURL string) (*Prospect, bool) {
	return findProspect(prospectURL)
}

// SetPassword : Set prospect password
func (p *Prospect) SetPassword() (string, bool) {
	c := ldap.Connect()
	return ldap.SetProspectPassword(c, p.URL)
}

// SaveScore : Save the prospoect score
func (p *Prospect) SaveScore(score *ProspectScore) bool {
	log.Printf("debug: creating score entry for %s", p.URL)
	err := connection.QueryRow(`
	INSERT INTO prospect_score
	(prospect_id, score, solved, difficulty, time_taken_ms, total_time_ms, pct_time_taken, command_count, start_time, end_time)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id, created, modified
	`,
		p.ID,
		score.Score,
		score.Solved,
		score.Difficulty,
		score.TimeTakenMS,
		score.TotalTimeMS,
		score.PctTimeTaken,
		score.CommandCount,
		score.StartTime,
		score.EndTime,
	).Scan(&score.ID, &score.Created, &score.Modified)
	if err != nil {
		log.Printf("error: failed to create score for %s: %s", p.URL, err)
		return false
	}
	return true
}

// GetScore : Get the score for a prospect
func (p *Prospect) GetScore() (*ProspectScore, bool) {
	log.Printf("debug: getting score for %s", p.URL)
	var score ProspectScore
	err := connection.QueryRow(`
	SELECT
	id, prospect_id, score, solved, difficulty, time_taken_ms, total_time_ms, pct_time_taken, command_count, start_time, end_time, created, modified
	FROM prospect_score
	WHERE prospect_id = $1`, p.ID,
	).Scan(
		&score.ID,
		&p.ID,
		&score.Score,
		&score.Solved,
		&score.Difficulty,
		&score.TimeTakenMS,
		&score.TotalTimeMS,
		&score.PctTimeTaken,
		&score.CommandCount,
		&score.StartTime,
		&score.EndTime,
		&score.Created,
		&score.Modified,
	)
	if err != nil {
		log.Printf("error: failed to find score for %s: %s", p.URL, err)
		return nil, false
	}
	return &score, true
}

// SaveExamLog : Save the exam log for a prospect
func (p *Prospect) SaveExamLog(l string) (*ExamLog, bool) {
	log.Printf("debug: saving exam log for %s", p.URL)
	var examLog ExamLog
	err := connection.QueryRow(`
	INSERT INTO exam_log
	(prospect_id, log)
	VALUES ($1, $2)
	RETURNING created, modified`,
		p.ID, l,
	).Scan(
		&examLog.Created,
		&examLog.Modified,
	)
	if err != nil {
		log.Printf("error: failed to save exam log for %s: %s", p.URL, err)
		return nil, false
	}
	examLog.Log = l
	examLog.ProspectID = p.ID
	return &examLog, true
}

// SaveVetzukiLog : Save the vetzuki log for a prospect
func (p *Prospect) SaveVetzukiLog(l string) (*VetzukiLog, bool) {
	log.Printf("debug: saving vetzuki log for %s", p.URL)
	var vetzukiLog VetzukiLog
	err := connection.QueryRow(`
	INSERT INTO vetzuki_log
	(prospect_id, log)
	VALUES ($1, $2)
	RETURNING created, modified`,
		p.ID, l,
	).Scan(
		&vetzukiLog.Created,
		&vetzukiLog.Modified,
	)
	if err != nil {
		log.Printf("error: failed to save vetzuki log for %s: %s", p.URL, err)
		return nil, false
	}
	vetzukiLog.Log = l
	vetzukiLog.ProspectID = p.ID
	return &vetzukiLog, true
}

// GetEmployer : Get an employer by ID
func GetEmployer(id int64) (*Employer, bool) {
	return findEmployerByID(id)
}

// GetEmployerByEmail : Get an employer by email
func GetEmployerByEmail(email string) (*Employer, bool) {
	return findEmployer(email)
}
func createEmployerProspect(employerExam *EmployerExam, prospect *Prospect) (*EmployerProspect, bool) {
	log.Printf("debug: exam %d for %s", employerExam.ExamID, prospect.Email)
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
	log.Printf("debug: query result returned")

	if row == nil {
		log.Printf("error: failed to create employerProspect %d,%d,%d: %v",
			ep.ProspectID, ep.EmployerID, ep.EmployerExamID, row)
		return nil, false
	}
	return ep, true
}
func createProspect(employerExam *EmployerExam, name, email, role string) (*Prospect, bool) {
	log.Printf("debug: creating exam %d for %s", employerExam.ExamID, email)

	employer, ok := GetEmployer(employerExam.EmployerID)
	if !ok {
		log.Printf("error: unable to locate employer %d", employerExam.EmployerID)
		return nil, false
	}
	prospect := &Prospect{
		Name:           name,
		Email:          email,
		URL:            xid.New().String(),
		Role:           role,
		EmployerName:   employer.Name,
		EmployerID:     employerExam.EmployerID,
		EmployerExamID: employerExam.ExamID,
	}
	log.Printf("debug: prepared prospect %s", prospect.Email)
	err := connection.QueryRow(`
	  INSERT into prospect (name, email, url, employer_name, role, employer_id, employer_exam_id)
	  VALUES ($1, $2, $3, $4, $5, $6, $7)
	  RETURNING id, created, modified`,
		prospect.Name,
		prospect.Email,
		prospect.URL,
		prospect.EmployerName,
		prospect.Role,
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
	log.Printf("debug: creating for exam %s for employer %s", exam.Name, employer.Email)
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
	log.Printf("debug: finding prospect %d", id)
	prospect := &Prospect{ID: id}
	err := connection.QueryRow(`
	  SELECT name, email, url, role, employer_name, employer_id, employer_exam_id, created, modified
	  FROM prospect
	  WHERE id = $1
	`, id).Scan(
		&prospect.Name,
		&prospect.Email,
		&prospect.URL,
		&prospect.Role,
		&prospect.EmployerName,
		&prospect.EmployerID,
		&prospect.EmployerExamID,
		&prospect.Created,
		&prospect.Modified,
	)
	if err != nil {
		log.Printf("error: failed to locate prospect %d: %s", id, err)
		return nil, false
	}
	return prospect, true
}

const (
	// ScreeningStateUnconfirmed : Employer create, email link unfollowed
	ScreeningStateUnconfirmed = 0
	// ScreeningStateConfirmed : Link redeemed
	ScreeningStateConfirmed = 1
	// ScreeningStateActive : Login completed
	ScreeningStateActive = 2
	// ScreeningStateComplete : Teesh shell exited
	ScreeningStateComplete = 3
)

// SetScreeningState : Set the screening state of a prospect
func (p *Prospect) SetScreeningState(state int) bool {
	log.Printf("debug: setting %s state to %d", p.URL, state)
	p.ScreeningState = state
	err := connection.QueryRow(`
	UPDATE prospect
	SET screening_state = $1
	WHERE url = $2
	RETURNING id`,
		p.ScreeningState,
		p.URL).Scan(&p.ID)
	if err != nil {
		log.Printf("error: updating %s screening state to %d: %s", p.URL, state, err)
		return false
	}
	return true
}

// FindProspectScores : Find prospects and attach scores
func (e *Employer) FindProspectScores() ([]*ProspectWithScore, bool) {
	prospects, ok := e.FindProspects()
	if !ok {
		log.Printf("error: unable to find prospects for employer %s", e.Email)
		return nil, false
	}
	var prospectsWithScores []*ProspectWithScore
	for _, prospect := range prospects {
		withScore := &ProspectWithScore{
			Prospect: prospect,
			Score:    nil,
		}
		if prospect.ScreeningState == ScreeningStateComplete {
			score, ok := prospect.GetScore()
			if ok {
				withScore.Score = score
			}
		}
		prospectsWithScores = append(prospectsWithScores, withScore)
	}
	return prospectsWithScores, true
}

// FindProspects : Find prospects associated with the Employer
func (e *Employer) FindProspects() ([]*Prospect, bool) {
	log.Printf("debug: finding %s prospects", e.Email)
	rows, err := connection.Query(`
	select id, name, email, url, employer_id, created, modified, role, screening_state
	FROM prospect
	WHERE employer_id = $1`, e.ID)
	if err != nil {
		log.Printf("error: failed to find prospects for employer %s: %s", e.Email, err)
		return nil, false
	}
	defer rows.Close()
	prospects := []*Prospect{}
	for rows.Next() {
		prospect := &Prospect{}
		err := rows.Scan(
			&prospect.ID,
			&prospect.Name,
			&prospect.Email,
			&prospect.URL,
			&prospect.EmployerID,
			&prospect.Created,
			&prospect.Modified,
			&prospect.Role,
			&prospect.ScreeningState,
		)
		if err != nil {
			log.Printf("error: unable to scan into prospect: %s", err)
			return nil, false
		}
		prospects = append(prospects, prospect)
	}
	return prospects, true
}
func findProspect(prospectURL string) (*Prospect, bool) {
	log.Printf("debug: finding prospect %s", prospectURL)
	prospect := &Prospect{URL: prospectURL}
	err := connection.QueryRow(`
	SELECT id,name, email, role, screening_state, employer_name, employer_id, employer_exam_id, created, modified
	FROM prospect
	WHERE url = $1`, prospectURL).Scan(
		&prospect.ID,
		&prospect.Name,
		&prospect.Email,
		&prospect.Role,
		&prospect.ScreeningState,
		&prospect.EmployerName,
		&prospect.EmployerID,
		&prospect.EmployerExamID,
		&prospect.Created,
		&prospect.Modified,
	)
	if err != nil {
		log.Printf("error : failed to locate prospect %s: %s", prospectURL, err)
		return nil, false
	}
	log.Printf("debug: found prospect %s by %s", prospect.Email, prospect.URL)
	return prospect, true
}
func findExamByID(id int64) (*Exam, bool) {
	log.Printf("debug: finding %d", id)
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
	log.Printf("debug: finding %d", id)
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
	log.Printf("debug: finding %s", email)
	employer := &Employer{Email: email}
	row := connection.QueryRow(`
	  SELECT id, name, email,billing_email, billing_state, created, modified
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
		&employer.Email,
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

// ExamLog : Actual log of an exam
type ExamLog struct {
	ProspectID int64     `sql:"prospect_id" json:"prospectID"`
	Log        string    `sql:"log" json:"log"` // base64 encoded
	Created    time.Time `sql:"created" json:"created"`
	Modified   time.Time `sql:"modified" json:"modified"`
}

// VetzukiLog : Log of exam setup and teardown
type VetzukiLog struct {
	ProspectID int64     `sql:"prospect_id" json:"prospectID"`
	Log        string    `sql:"log" json:"log"` // base64 encoded
	Created    time.Time `sql:"created" json:"created"`
	Modified   time.Time `sql:"modified" json:"modified"`
}

// Organization : Container of Employers
type Organization struct {
	ID       int64     `sql:"id" json:"id"`
	Name     string    `sql:"name" json:"name"`
	Created  time.Time `sql:"created" json:"created"`
	Modified time.Time `sql:"modified" json:"modified"`
}

// OrganizationalRole : Role of Employer in Organization
type OrganizationalRole struct {
	EmployerID     int64     `sql:"employer_id" json:"employerID"`
	OrganizationID int64     `sql:"organization_id" json:"organizationID"`
	Role           int       `sql:"role" json:"role"`
	Created        time.Time `sql:"created" json:"created"`
	Modified       time.Time `sql:"modified" json:"modified"`
}

// ProspectScore : Result of a scoring
type ProspectScore struct {
	ID            int64     `sql:"id" json:"id"`
	ProspectURLID string    `json:"prospectURLID"`
	Score         float64   `sql:"score" json:"score"`
	Solved        float64   `sql:"solved" json:"solved"`
	TimeTakenMS   float64   `sql:"time_taken_ms" json:"timeTakenMS"`
	TotalTimeMS   float64   `sql:"total_time_ms" json:"totalTimeMS"`
	PctTimeTaken  float64   `sql:"pct_time_taken" json:"pctTimeTaken"`
	CommandCount  float64   `sql:"total_command_count" json:"totalCommandCount"`
	Difficulty    float64   `sql:"difficulty" json:"difficulty"`
	StartTime     time.Time `sql:"start_time" json:"startTime"`
	EndTime       time.Time `sql:"end_time" json:"endTime"`
	Created       time.Time `sql:"created" json:"created"`
	Modified      time.Time `sql:"modified" json:"modified"`
}

// ProspectWithScore : Prospect with scoring data
type ProspectWithScore struct {
	Prospect *Prospect      `json:"prospect"`
	Score    *ProspectScore `json:"score"`
}

// Employer : An employer can hire prospects
type Employer struct {
	ID             int64     `sql:"id" json:"id"`
	Created        time.Time `sql:"created" json:"created"`
	Modified       time.Time `sql:"modified" json:"modified"`
	Name           string    `sql:"name" json:"name"`
	Email          string    `sql:"email" json:"email"`
	BillingEmail   string    `sql:"billing_email" json:"billingEmail"`
	BillingID      int64     `sql:"billing_id" json:"billingID"`
	BillingState   int       `sql:"billing_state" json:"billingState"`
	OrganizationID int64     `sql:"organization_id" json:"organizationID"`
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
	Role           string    `sql:"role" json:"role"`
	EmployerName   string    `sql:"employer_name" json:"employerName"`
	ScreeningState int       `sql:"screening_state" json:"screeningState"`
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
	*Prospect      `json:"prospect"`
	*Employer      `json:"employer"`
}
