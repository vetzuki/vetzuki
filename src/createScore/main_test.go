package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

var (
	testExamLog = `2019-10-05 18:08:36.146262 Assign the IP 10.0.1.5/24 to the eth0 interface
2019-10-05 18:08:36.146397 ---
2019-10-05 18:08:43.727132 ]0;root@ad02fa9b7a19: ~root@ad02fa9b7a19:~# ip a a 10.0.1.5/24 dev eth0
2019-10-05 18:08:44.762010 ]0;root@ad02fa9b7a19: ~root@ad02fa9b7a19:~#                                                                                

2019-10-05 18:08:44.762091 Exam complete. See https://www.poc.vetzuki.com/p/bmcdkna43drs72ai6b20 for next steps.

2019-10-05 18:08:44.762118                                                                                

2019-10-05 18:08:49.872642                                                                                

2019-10-05 18:08:49.872721 Test will now exit...

2019-10-05 18:08:49.872735                                                                                

`
)

func TestStripTimestamp(t *testing.T) {
	type testCase struct {
		input, timestamp, message string
		err                       error
	}
	testCases := []testCase{
		testCase{
			input:     "2019-10-05 20:43:46.123456 Welcome to Ubuntu 18.04.3 LTS (GNU/Linux 4.14.146-119.123.amzn2.x86_64 x86_64)",
			timestamp: "2019-10-05 20:43:46.123456",
			message:   "Welcome to Ubuntu 18.04.3 LTS (GNU/Linux 4.14.146-119.123.amzn2.x86_64 x86_64)",
			err:       nil,
		},
		testCase{
			input:     "2019-10-05 20:43:46.123456",
			timestamp: "2019-10-05 20:43:46.123456",
			message:   "",
			err:       nil,
		},
	}
	for _, testCase := range testCases {
		timestamp, message, err := StripTimestamp(testCase.input)
		if err != nil {
			t.Fatalf("expected no errors, got %s", err)
		}
		if timestamp.String() != testCase.timestamp+" +0000 UTC" {
			t.Fatalf("expected timestamp %s, got %s", testCase.timestamp, timestamp.String())
		}
		if message != testCase.message {
			t.Fatalf("expected message %s, got %s", testCase.message, message)
		}
	}

}

func TestAssignment(t *testing.T) {
	a := getAssignment(testExamLog)
	if len(a) != 2 {
		t.Fatalf("expected a 2 line assignment, got %d lines", len(a))
	}
}
func TestExamLog(t *testing.T) {
	l := getExamLog(testExamLog)
	if len(l) != 2 {
		t.Fatalf("expected a 2 line exam log, got %d lines: %#v", len(l), l)
	}
}

func TestDeserializePOST(t *testing.T) {
	examLogBase64, err := ioutil.ReadFile("./exam.log.base64")
	if err != nil {
		t.Fatalf("unable to read exam.log.base64")
	}
	raw := fmt.Sprintf(`{
		"prospectURLID": "url",
		"examLog": "%s"
	}`, strings.TrimSuffix(string(examLogBase64), "\n"))
	fmt.Println(raw)
	var s ScoringRequest
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("expected to unmarshal but failed, %s", err)
	}

}
