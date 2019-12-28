package logger_test

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/macrat/landns/lib-landns/logger"
)

func TestLevel(t *testing.T) {
	tests := []struct {
		Level  logger.Level
		Expect string
	}{
		{logger.DebugLevel, "debug"},
		{logger.InfoLevel, "info"},
		{logger.WarnLevel, "warning"},
		{logger.ErrorLevel, "error"},
		{logger.FatalLevel, "fatal"},
	}

	for _, tt := range tests {
		if tt.Level.String() != tt.Expect {
			t.Errorf("failed to convert to human readable string: expected %#v but got %#v", tt.Expect, tt.Level.String())
		}
	}
}

type LoggerTestEntry struct {
	Level string
	Func  func(string, logger.Fields)
	Exit  bool
}

type LoggerTest []LoggerTestEntry

func (tests LoggerTest) Run(t *testing.T, buf *bytes.Buffer, l *logger.BasicLogger) {
	exit := false
	l.Logger.ExitFunc = func(int) {
		exit = true
	}

	for _, tt := range tests {
		buf.Reset()
		exit = false
		tt.Func("hello", logger.Fields{
			"target": "world",
			"id":     1,
		})

		expect := fmt.Sprintf(`time="20..-..-..T..:..:..+..:.." level=%s msg=hello id=1 target=world\n`, tt.Level)

		if ok, err := regexp.MatchString("^"+expect+"$", buf.String()); err != nil {
			t.Errorf("failed to compare: %s", err)
		} else if !ok {
			t.Errorf("failed to logging %s:\nexpected: %#v\nbut got:  %#v", tt.Level, expect, buf.String())
		}

		if exit != tt.Exit {
			t.Errorf("unexpected exit status: expected %v but got %v", tt.Exit, exit)
		}
	}
}

func TestBasicLogger(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	l := logger.New(buf, logger.DebugLevel)

	LoggerTest{
		{"debug", l.Debug, false},
		{"info", l.Info, false},
		{"warning", l.Warn, false},
		{"error", l.Error, false},
		{"fatal", l.Fatal, true},
	}.Run(t, buf, l)
}

func TestDefaultLogger(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	l := logger.New(buf, logger.DebugLevel)
	logger.SetLogger(l)

	if logger.GetLogger() != l {
		t.Fatalf("failed to set/get default logger")
	}

	LoggerTest{
		{"debug", logger.Debug, false},
		{"info", logger.Info, false},
		{"warning", logger.Warn, false},
		{"error", logger.Error, false},
		{"fatal", logger.Fatal, true},
	}.Run(t, buf, l)
}
