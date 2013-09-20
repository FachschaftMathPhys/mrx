package log

import (
	"code.google.com/p/go.net/websocket"
	"container/list"
	"fmt"
	"time"
	"os"
)

type LogType int
const (
	LogNone	LogType = iota
	LogWarning
	LogError
	LogFatal
)

type LogLine struct {
	Time	time.Time
	Type	LogType
	Text	string
}

type ClientState int
const (
	StateNew ClientState = iota
	StateAlive
	StateDead
)

type LogClient struct {
	Channel	chan LogLine
	Socket	*websocket.Conn
	Close	chan int
}

var logClients list.List
var logLines   []LogLine

func PutLine(line LogLine) {
	fmt.Fprintf(os.Stderr, "%s\n", line)
	for e := logClients.Front(); e != nil; e = e.Next() {
		client := e.Value.(*LogClient)
		client.Channel <- line
	}
}

func LogServer(ws *websocket.Conn) {
	Print("New log connection")
	client := LogClient{ make(chan LogLine), ws, make(chan int) }
	logClients.PushBack(&client)
	for _, line := range(logLines) {
		websocket.JSON.Send(ws, line)
	}
	for line := range(client.Channel) {
		websocket.JSON.Send(ws, line)
		if line.Type == LogFatal {
			ws.Close()
		}
	}
	for e := logClients.Front(); e != nil; e = e.Next() {
		c := e.Value.(*LogClient)
		if c == &client {
			logClients.Remove(e)
			break
		}
	}
	client.Close <- 1
}

func (l *LogLine) String() string {
	var typ string
	switch (l.Type) {
	case LogNone:
		typ = " "
	case LogWarning:
		typ = "W"
	case LogError:
		typ = "E"
	}
	return fmt.Sprintf("%s %s\t%s", l.Time, typ, l.Text)
}

func (t LogType) String() string {
	switch (t) {
	case LogNone:
		return ""
	case LogWarning:
		return "warning"
	case LogError:
		return "error"
	case LogFatal:
		return "fatal"
	}
	return "?"
}

func Warn(v ...interface{}) {
	line := LogLine{ time.Now(), LogWarning, fmt.Sprint(v...) }
	logLines = append(logLines, line)
	PutLine(line)
}

func Warnf(s string, v ...interface{}) {
	line := LogLine{ time.Now(), LogWarning, fmt.Sprintf(s, v...) }
	logLines = append(logLines, line)
	PutLine(line)
}

func Print(v ...interface{}) {
	line := LogLine{ time.Now(), LogNone, fmt.Sprint(v...) }
	logLines = append(logLines, line)
	PutLine(line)
}

func Printf(s string, v ...interface{}) {
	line := LogLine{ time.Now(), LogNone, fmt.Sprintf(s, v...) }
	logLines = append(logLines, line)
	PutLine(line)
}

func Error(v ...interface{}) {
	line := LogLine{ time.Now(), LogError, fmt.Sprint(v...) }
	logLines = append(logLines, line)
	PutLine(line)
}

func Errorf(s string, v ...interface{}) {
	line := LogLine{ time.Now(), LogError, fmt.Sprintf(s, v...) }
	logLines = append(logLines, line)
	PutLine(line)
}

func Fatal(v ...interface{}) {
	line := LogLine{ time.Now(), LogFatal, fmt.Sprint(v...) }
	logLines = append(logLines, line)
	PutLine(line)
	for e := logClients.Front(); e != nil; e = e.Next() {
		client := e.Value.(*LogClient)
		close(client.Channel)
	}

	os.Exit(1)
}
