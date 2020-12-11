//+build windows

package logger

import (
	"bytes"
	"encoding/xml"
	"log"
	"os/exec"
	"testing"
	"time"

	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Levels int

const (
	Info Levels = iota + 1
	Warning
	Error
)

type Event struct {
	Message string `xml:"EventData>Data"`
	Level   Levels `xml:"System>EventID"`
}

func getEventLog(t *testing.T, since time.Time) []Event {
	timeStr := since.UTC().Format(time.RFC3339)
	cmd := exec.Command("wevtutil", "qe", "Application", "/rd:true", "/q:Event[System[TimeCreated[@SystemTime >= '"+timeStr+"'] and Provider[@Name='Telegraf']]]")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	require.NoError(t, err)
	xmlStr := "<events>" + out.String() + "</events>"
	var events struct {
		Events []Event `xml:"Event"`
	}
	err = xml.Unmarshal([]byte(xmlStr), &events)
	require.NoError(t, err)
	return events.Events
}

func TestEventLog(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	prepareLogger(t)

	config := LogConfig{
		LogTarget: LogTargetEventlog,
		Logfile:   "",
	}

	SetupLogging(config)
	now := time.Now()
	log.Println("I! Info message")
	log.Println("W! Warn message")
	log.Println("E! Err message")
	events := getEventLog(t, now)
	assert.Len(t, events, 3)
	assert.Contains(t, events, Event{Message: "Info message", Level: Info})
	assert.Contains(t, events, Event{Message: "Warn message", Level: Warning})
	assert.Contains(t, events, Event{Message: "Err message", Level: Error})
}

func TestRestrictedEventLog(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in  short mode")
	}
	prepareLogger(t)

	config := LogConfig{
		LogTarget: LogTargetEventlog,
		Quiet:     true,
	}

	SetupLogging(config)
	//separate previous log messages by small delay
	time.Sleep(time.Second)
	now := time.Now()
	log.Println("I! Info message")
	log.Println("W! Warning message")
	log.Println("E! Error message")
	events := getEventLog(t, now)
	assert.Len(t, events, 1)
	assert.Contains(t, events, Event{Message: "Error message", Level: Error})
}

func prepareLogger(t *testing.T) {
	svc, err := service.New(nil, &service.Config{Name: "Telegraf"})
	require.NoError(t, err)
	svcLogger, err := svc.SystemLogger(nil)
	require.NoError(t, err)
	require.NotNil(t, svcLogger)
	registerLogger(LogTargetEventlog, &eventLoggerCreator{serviceLogger: svcLogger})
}
