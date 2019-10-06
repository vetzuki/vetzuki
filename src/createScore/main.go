package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vetzuki/vetzuki/auth"
	lambdaEvents "github.com/vetzuki/vetzuki/events"
	"github.com/vetzuki/vetzuki/model"
	"log"
	"regexp"
	"strings"
	"time"
)

const (
	examTime30Minutes = float64(60 * 30)
	examDuration      = time.Millisecond * (60 * 30 * 1000)
	examDifficulty    = 0.25
)

var (
	// Solved exam
	regexSolvedExam = regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]{6} Exam complete. See https.*")
	// Time limit exceeded
	regexTimeLimitExceeded = regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]{6} Exam time limit exceeded. See https.*")
)

// ScoringRequest : An encoded scoring request
type ScoringRequest struct {
	ProspectURLID    string `json:"prospectURLID"`
	ExamLogBase64    string `json:"examLog"`
	VetzukiLogBase64 string `json:"vetzukiLog"`
}

// Exam : Structure of exam
type Exam struct {
	Assignment      []string  `json:"assignment"`
	Log             []string  `json:"log"`
	ProctorMessages []string  `json:"proctorMessages"`
	StartTime       time.Time `json:"startTime"`
	EndTime         time.Time `json:"endTime"`
}

// ScoreProspect : Score the prospect based on their performance
func ScoreProspect(s ScoringRequest) (*model.ProspectScore, bool) {
	log.Printf("debug: scoring exam for %s", s.ProspectURLID)
	rawLog, err := base64.StdEncoding.DecodeString(s.ExamLogBase64)
	if err != nil {
		log.Printf("error: unable to decode exam log: %s", err)
		return nil, false
	}
	prospect, ok := model.GetProspect(s.ProspectURLID)
	if !ok {
		log.Printf("error: unable to locate prospect %s", s.ProspectURLID)
		return nil, false
	}
	// When processing the exam log, watch out for empty command lines
	// they should not count toward the command total
	assignment := getAssignment(string(rawLog))
	examLog := getExamLog(string(rawLog))
	startTime, _, err := StripTimestamp(assignment[len(assignment)-1])
	if err != nil {
		log.Printf("error: unable to get exam start time")
		return nil, false
	}
	endTime, ok := getEndTime(string(rawLog))
	if !ok {
		log.Printf("error: unable to get exam end time")
		return nil, false
	}
	timeTaken := endTime.Sub(startTime)
	log.Printf("debug: exam took %f seconds", timeTaken.Seconds())
	pctTimeTaken := float64(timeTaken.Milliseconds()) / float64(examDuration.Milliseconds())
	log.Printf("debug: exam completed in %%%2f of total time", pctTimeTaken)
	totalCommands := float64(len(examLog))
	// When a blank command is at the command line before the broadcast
	// message, do not count the line in the total
	if strings.HasSuffix(examLog[len(examLog)-1], "#") {
		totalCommands--
	}
	solved := getExamOutcome(string(rawLog))

	score := solved - (pctTimeTaken * ((float64(1) / totalCommands) / examDifficulty))
	log.Printf("debug: %s scored %3f", s.ProspectURLID, score)
	prospectScore := &model.ProspectScore{
		ProspectURLID: s.ProspectURLID,
		Score:         score,
		Solved:        solved,
		Difficulty:    examDifficulty,
		TimeTakenMS:   float64(timeTaken.Milliseconds()),
		TotalTimeMS:   float64(examDuration.Milliseconds()),
		PctTimeTaken:  pctTimeTaken,
		CommandCount:  totalCommands,
		StartTime:     startTime,
		EndTime:       endTime,
	}
	_, ok = prospect.SaveExamLog(s.ExamLogBase64)
	if !ok {
		log.Printf("warning: failed to save exam log for %s", s.ProspectURLID)
	}
	_, ok = prospect.SaveVetzukiLog(s.VetzukiLogBase64)
	if !ok {
		log.Printf("warning: failed to save vetzuki log for %s", s.ProspectURLID)
	}
	if !prospect.SaveScore(prospectScore) {
		log.Printf("error: failed to save score for prospect %s", s.ProspectURLID)
		return nil, false
	}
	return prospectScore, true
}

func getAssignment(examLog string) []string {
	assignment := strings.SplitN(examLog, "---", 2)
	return strings.Split(assignment[0], "\n")
}
func getExamLog(examLog string) []string {
	lines := strings.SplitN(examLog, "---", 2)
	var logLines []string
	for _, line := range strings.Split(lines[1], "\n") {
		if regexSolvedExam.MatchString(line) || regexTimeLimitExceeded.MatchString(line) {
			break
		}
		if len(line) != 0 {
			logLines = append(logLines, line)
		}
	}
	return logLines
}

const (
	examSolved         = float64(1)
	examTimedOut       = 0
	examOutcomeUnknown = 0
)

func getExamOutcome(examLog string) float64 {
	lines := strings.SplitN(examLog, "---", 2)
	for _, line := range strings.Split(lines[1], "\n") {
		if regexSolvedExam.MatchString(line) {
			return examSolved
		}
		if regexTimeLimitExceeded.MatchString(line) {
			return examTimedOut
		}
	}
	log.Printf("warning: exam outcome unknown")
	return examOutcomeUnknown
}
func getEndTime(examLog string) (time.Time, bool) {
	lines := strings.SplitN(examLog, "---", 2)
	for _, line := range strings.Split(lines[1], "\n") {
		if regexSolvedExam.MatchString(line) || regexTimeLimitExceeded.MatchString(line) {
			t, _, err := StripTimestamp(line)
			if err != nil {
				log.Printf("error: unable to get end time from matched line: %s", err)
				return time.Time{}, false
			}
			return t, true
		}
	}
	return time.Time{}, false
}

// StripTimestamp : Split the string on the timestamp
func StripTimestamp(logLine string) (time.Time, string, error) {
	r := strings.SplitN(logLine, " ", 3)
	switch len(r) {
	case 3:
		timestamp := fmt.Sprintf("%s %s", r[0], r[1])
		t, err := ToTime(timestamp, timeLayout)
		if err != nil {
			log.Printf("error: unable to parse timestamp %s: %s", timestamp, err)
			return time.Time{}, "", err
		}
		return t, r[2], err
	case 2:
		timestamp := fmt.Sprintf("%s %s", r[0], r[1])
		t, err := ToTime(timestamp, timeLayout)
		if err != nil {
			log.Printf("error: unable to parse timestamp %s: %s", timestamp, err)
			return time.Time{}, "", err
		}
		return t, "", err
	default:
		log.Printf("error: unable to partition log line")
	}
	return time.Time{}, "", fmt.Errorf("unable to locate timestamp")
}

const timeLayout = "2006-01-02 15:04:05.999999" // 6-precision
// ToTime : Convert a log timestamp to a time
func ToTime(timestamp string, layout string) (time.Time, error) {
	return time.Parse(layout, timestamp)
}

// Handler : Create a score
func Handler(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	apiKey, ok := auth.ExtractToken(r.Headers)
	if !ok {
		log.Printf("error: missing api key")
		return lambdaEvents.AccessDenied, nil
	}
	if !auth.ValidateAPIKey(apiKey) {
		log.Printf("error: api key is not authorized")
		return lambdaEvents.AccessDenied, nil
	}
	var scoringRequest ScoringRequest
	err := json.Unmarshal([]byte(r.Body), &scoringRequest)
	if err != nil {
		log.Printf("error: failed to unmarshal: %s", err)
		return lambdaEvents.ServerError, err
	}
	log.Printf("debug: submitting scoring request for %s", scoringRequest.ProspectURLID)
	prospectScore, ok := ScoreProspect(scoringRequest)
	if !ok {
		log.Printf("error: failed to score prospect %s", scoringRequest.ProspectURLID)
		return lambdaEvents.ServerError, fmt.Errorf("failed to score prospect")
	}
	prospectScoreJSON, err := json.Marshal(prospectScore)
	if err != nil {
		log.Printf("error: serializing prospect score failed: %s", err)
		return lambdaEvents.ServerError, err
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(prospectScoreJSON),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
