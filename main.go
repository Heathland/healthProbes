package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Server struct {
	port             int
	t                time.Time
	jobStart         time.Time
	waitStartupTime  time.Duration
	waitLivenessTime time.Duration
	jobDuration      time.Duration
}

func main() {
	var s Server
	s.port = 8080

	err := s.getEnvValues()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/startupProbe", s.startupProbe)
	http.HandleFunc("/livenessProbe", s.livenessProbe)
	http.HandleFunc("/readinessProbe", s.readinessProbe)
	http.HandleFunc("/startJob", s.startJob)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Start time
	s.t = time.Now()

	fmt.Printf("Starting server. Listening on port: %d", s.port)
	log.Fatal(srv.ListenAndServe())
}

func getEnvToDuration(e string) (d time.Duration, err error) {
	var envValue int
	envValue, err = strconv.Atoi(os.Getenv(e))
	d = time.Duration(envValue) * time.Second
	return
}

func (s *Server) getEnvValues() (err error) {
	s.waitStartupTime, err = getEnvToDuration("WAIT_STARTUP_TIME")
	if err != nil {
		return
	}
	s.waitLivenessTime, err = getEnvToDuration("WAIT_LIVENESS_TIME")
	if err != nil {
		return
	}
	s.jobDuration, err = getEnvToDuration("JOB_DURATION_TIME")
	if err != nil {
		return
	}
	return
}

func (s *Server) startupProbe(w http.ResponseWriter, r *http.Request) {
	if time.Since(s.t) > s.waitStartupTime {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}
}

func (s *Server) livenessProbe(w http.ResponseWriter, r *http.Request) {
	if time.Since(s.t) > s.waitLivenessTime {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}
}

func (s *Server) readinessProbe(w http.ResponseWriter, r *http.Request) {
	if time.Since(s.jobStart) > s.jobDuration {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}
}

func (s *Server) startJob(w http.ResponseWriter, r *http.Request) {
	if time.Since(s.jobStart) > s.jobDuration {
		s.jobStart = time.Now()
		fmt.Fprintf(w, "Pod (%s)\nStarting job. Unavailable till: %s", os.Getenv("HOSTNAME"), s.jobStart.Add(s.jobDuration).Format("Mon Jan _2 15:04:05 2006"))
	} else {
		fmt.Fprintf(w, "Still running job. Unavailable till: %v", s.jobStart.Add(s.jobDuration))
	}
}
