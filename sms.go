package main

import (
	"crypto/sha1"
	"code.google.com/p/go.net/websocket"
	"container/list"
	"encoding/json"
	"fmt"
	"git.merovius.de/bulkSMS"
    "git.merovius.de/MrX/log"
	"github.com/ziutek/gogammu"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type SMS struct {
	Time	time.Time
	Number	string
	Body	string
}

var Incoming	[]SMS
var stick		*gammu.StateMachine
var sender		*bulkSMS.BulkSMS
var Registered	[]string
var smsClients	list.List
var regClients  list.List
var reRegister	*regexp.Regexp = regexp.MustCompile("(?i)^register")
var NumMrX		string

type sortableSMS []SMS

type Registration struct {
	Number string
}

func (s sortableSMS) Len() int {
	return len(s)
}

func (s sortableSMS) Less(i, j int) bool {
	return s[i].Time.Before(s[j].Time)
}

func (s sortableSMS) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func SendToClients(sms SMS) {
	log.Print("SendToClients")
	for e := smsClients.Front(); e != nil; e = e.Next() {
		c := e.Value.(chan SMS)
		c <- sms
	}
}

func forwardSMS(ws *websocket.Conn, input chan SMS) {
	for sms := range(input) {
		websocket.JSON.Send(ws, &sms)
	}
}

func forwardReg(ws *websocket.Conn, input chan string) {
	for number := range(input) {
		websocket.JSON.Send(ws, []string{number})
	}
}

func SmsServer(ws *websocket.Conn) {
	log.Print("New sms connection")
	for _, sms := range(Incoming) {
		websocket.JSON.Send(ws, sms)
	}
	var sms SMS
	c := make(chan SMS)
	go forwardSMS(ws, c)
	e := smsClients.PushBack(c)
	for {
		err := websocket.JSON.Receive(ws, &sms)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			continue
		}
		var recipients []string
		if sms.Number == "all" {
			recipients = Registered
		} else {
			recipients = []string{ sms.Number }
		}
		s := bulkSMS.NewSMS(sms.Body, recipients)
		err = sender.Send(s)
		if err != nil {
			log.Error("Could not send sms: ", err)
			continue
		}
		log.Printf("Sending SMS to %s: %s", sms.Number, s.Status())

		credits, err := sender.GetCredits()
		if err != nil {
			log.Warn("Could not get credits: ", err)
		} else {
			log.Printf("%v Credits left", credits)
		}
	}
	close(c)
	smsClients.Remove(e)
}

func RegServer(ws *websocket.Conn) {
	log.Print("New reg connection")
	err := websocket.JSON.Send(ws, Registered)
	if err != nil {
		log.Error(err)
	}
	var reg Registration
	c := make(chan string)
	go forwardReg(ws, c)
	e := regClients.PushBack(c)
	for {
		err = websocket.JSON.Receive(ws, &reg)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			continue
		}
		if strings.HasPrefix(reg.Number, "+") {
			reg.Number = reg.Number[1:]
		} else if strings.HasPrefix(reg.Number, "0") {
			reg.Number = "49" + reg.Number[1:]
		}
		Registered = append(Registered, reg.Number)
		log.Printf("Registration from %s", reg.Number)
		websocket.JSON.Send(ws, [...]string{reg.Number})
		PutRegistrations()
	}
	close(c)
	regClients.Remove(e)
}

func LoadArchive() {
	files, err := filepath.Glob("sms_archive/*")
	if err != nil {
		log.Error("Could not glob archive:", err)
		return
	}
	archive := make(sortableSMS, len(files))
	for i, file := range(files) {
		f, err := os.Open(file)
		if err != nil {
			log.Errorf("Could not open file %s:", file, err)
			continue
		}
		dec := json.NewDecoder(f)
		var sms SMS
		if err := dec.Decode(&sms); err != nil {
			log.Errorf("Could not parse file %s:", file, err)
		}
		archive[i] = sms
	}
	log.Printf("Loaded %d sms from archive", len(archive))
	sort.Sort(archive)
	Incoming = archive
}

func Archive(sms SMS) {
	sha := sha1.New()
	text, err := json.Marshal(sms)
	if err != nil {
		log.Error("Could not encode sms:", err)
	}
	sha.Write(text)
	sum := fmt.Sprintf("%x", sha.Sum(nil))
	f, err := os.OpenFile(path.Join("sms_archive", sum), os.O_RDWR | os.O_CREATE | os.O_TRUNC, 0666)
	if err != nil {
		log.Error("Could not open archive-file:", err)
		return
	}
	f.Write(text)
	f.Close()
}

func ReadSMS() {
	for {
		sms, err := stick.GetSMS()
		if err == io.EOF {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err != nil {
			log.Error("Could not get SMS:", err)
			continue
		}
		log.Print("New SMS:", sms)
		s := SMS{ sms.Time, sms.Number, sms.Body }
		Archive(s)
		SendToClients(s)
		Incoming = append(Incoming, s)
		if strings.HasPrefix(sms.Number, "+") {
			sms.Number = sms.Number[1:]
		} else if strings.HasPrefix(sms.Number, "0") {
			sms.Number = "49" + sms.Number[1:]
		}
		if reRegister.MatchString(sms.Body) {
			log.Print("Registration from ", sms.Number)
			for e := regClients.Front(); e != nil; e = e.Next() {
				c := e.Value.(chan string)
				c <- sms.Number
			}
			Registered = append(Registered, sms.Number)
			PutRegistrations()
		} else if sms.Number == NumMrX {
	        s := bulkSMS.NewSMS(sms.Body, Registered)
			err = sender.Send(s)
			if err != nil {
				log.Error("Could not send SMS: ", err)
			}
			log.Printf("Forwarding \"%s\" to all: %s", sms.Body, s.Status())

			credits, err := sender.GetCredits()
			if err != nil {
				log.Warn("Could not get credits: ", err)
			} else {
				log.Printf("%v Credits left", credits)
			}
		}
	}
}

func PutRegistrations() {
    text, err := json.Marshal(Registered)
    if err != nil {
        log.Error("Could not encode registrations: ", err)
    }
    f, err := os.OpenFile("registrations.json", os.O_RDWR | os.O_CREATE | os.O_TRUNC, 0666)
    if err != nil {
        log.Error("Could not open registrations-file: ", err)
        return
    }
    f.Write(text)
    f.Close()
	log.Printf("Saved %d registrations in registrations.json", len(Registered))
}

func GetRegistrations() {
	f, err := os.Open("registrations.json")
	if err != nil {
		log.Error("Could not open registrations-file: ", err)
		return
	}
	dec := json.NewDecoder(f)
	if err := dec.Decode(&Registered); err != nil {
		log.Error("Could not parse registrations-file: ", err)
	}
	log.Printf("Loaded %d numbers from registrations.json", len(Registered))
}

func InitSMS() {
	go LoadArchive()
	go GetRegistrations()
	var err error
	stick, err = gammu.NewStateMachine("")
	if err != nil {
		log.Fatal("Could not create StateMachine:", err)
	}
	err = stick.Connect()
	if err != nil {
		log.Fatal("Could not connect to stick:", err)
	}
	go ReadSMS()

	sender = bulkSMS.New("FSMathPhys", "wahsu3coZa8xaek1iw6g", "491606632613")

	credits, err := sender.GetCredits()
	if err != nil {
		log.Warn("Could not get credits: ", err)
	} else {
		log.Printf("%v credits left", credits)
	}
}
