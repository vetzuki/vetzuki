package model

import (
	"fmt"
	"github.com/vetzuki/vetzuki/ldap"
	"testing"
)

var testEnvironment = "development"

func init() {
	// configure LDAP
	baseDN := fmt.Sprintf("ou=prospects,dc=%s,dc=vetzuki,dc=com", testEnvironment)
	bindDN := fmt.Sprintf("cn=admin,dc=%s,dc=vetzuki,dc=com", testEnvironment)
	bindPassword := testEnvironment
	ldap.ConfigureConnection(baseDN, bindDN, bindPassword)
}

func TestFindEmployerByID(t *testing.T) {
	Connect()
	id := int64(1)
	e, ok := findEmployerByID(id)
	if !ok {
		t.Fatalf("expected to find %d, but failed", id)
	}
	if e.Email != "recruiter@employer1.com" {
		t.Fatalf("expected email addresses to match, got %s", e.Email)
	}
}
func TestFindEmployer(t *testing.T) {
	Connect()
	email := "recruiter@employer1.com"
	e, ok := findEmployer(email)
	if !ok {
		t.Fatalf("expected to find %s, but failed", email)
	}

	if e.Email != email {
		t.Fatalf("expected email %s, got %s", email, e.Email)
	}
}

func TestFindExamByID(t *testing.T) {
	Connect()
	e, ok := findExamByID(int64(1))
	if !ok {
		t.Fatalf("expected to find %d, but failed", int64(1))
	}
	if e.Name != "exam" {
		t.Fatalf("expected exam named 'exam', got %s", e.Name)
	}
}

func TestCreateEmployerExam(t *testing.T) {
	employer := &Employer{ID: int64(1)}
	exam := &Exam{ID: int64(1)}
	ee, ok := createEmployerExam(employer, exam)
	if !ok {
		t.Fatalf("expected to create employerExam but failed")
	}
	if ee.EmployerID != employer.ID {
		t.Fatalf("expected employer ID to be %d, got %d", employer.ID, ee.EmployerID)
	}
	if ee.ExamID != exam.ID {
		t.Fatalf("expected exam ID to be %d, got %d", exam.ID, ee.ExamID)
	}
}

func TestCreateProspect(t *testing.T) {
	employerExam, _ := createEmployerExam(
		&Employer{ID: int64(1)},
		&Exam{ID: int64(1)},
	)
	for i := range make([]int, 3) {
		name := fmt.Sprintf("prospect%d", i)
		email := fmt.Sprintf("%s@testmail.com", name)
		role := "role"
		p, ok := createProspect(employerExam, name, email, role)
		if !ok {
			t.Fatalf("expected to create prospect %s, but failed", email)
		}
		if p.Email != email {
			t.Fatalf("expected prospect email to be %s, got %s", email, p.Email)
		}
	}
}

func TestCreateEmployerProspect(t *testing.T) {
	employerExam, _ := createEmployerExam(
		&Employer{ID: int64(1)},
		&Exam{ID: int64(1)},
	)
	prospect, _ := createProspect(employerExam, "name", "email@email.com", "role")
	ep, ok := createEmployerProspect(employerExam, prospect)
	if !ok {
		t.Fatalf("expected to create employerProspect but failed")
	}
	if ep.ProspectID != prospect.ID {
		t.Fatalf("expected EmployerProspect.ProspectID to equal %d", prospect.ID)
	}
	if ep.EmployerID != employerExam.EmployerID {
		t.Fatalf("expected EmployerProspect.EmployerID to equal %d", employerExam.EmployerID)
	}
	if ep.EmployerExamID != employerExam.ID {
		t.Fatalf("expected EmployerProspect.EmployerExamID to equal %d, got %d", employerExam.ID, ep.EmployerExamID)
	}
}

func TestPublicCreateEmployerProspect(t *testing.T) {
	Connect()
	employerID := int64(1)
	examID := int64(1)
	name := "prospect jones"
	email := "prospect@email.com"
	role := "role"
	ep, ok := CreateEmployerProspect(employerID, examID, name, email, role)
	if !ok {
		t.Fatalf("expected to create EmployerProspect but failed")
	}
	prospect, ok := findProspectByID(ep.ProspectID)
	if !ok {
		t.Fatalf("expected linked ProspectID to be a Prospect")
	}
	if prospect.Email != email {
		t.Fatalf("expected prospect email to be %s, got %s", email, prospect.Email)
	}
}
