package centurylinkchallenge

import (
	"time"
	"github.com/gorilla/mux"
	"net/http"
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

/*
In this project, two web API endpoints are necessary. They are:

1. Record load for a given server.  This should take a:
- server name (string)
- CPU load (double)
- RAM load (double)And apply the values to an in-memory model used to provide the data in endpoint #2.

2. Display loads for a given server.  This should return data (if it has any) for the given server:
- A list of the average load values for the last 60 minutes broken down by minute
- A list of the average load values for the last 24 hours broken down by hour
Assume these endpoints will be under a continuous load being called for thousands of individual servers every minute.
*/

// The context of servers that have pulsed with updates
type Context struct {
	servers map[string]*server
}

// The server object
type server struct {
	Name       string
	Statistics []*pulse
}

// Allows capturing other information easily without changing the structure
type pulse struct {
	Name string         `json:"name"`
	Cpu  float64        `json:"cpu"`
	Mem  float64        `json:"mem"`
	Time time.Time      `json:"time"`
}

// Utilized when finding averages within a timeframe
type average struct {
	count  int
	memsum float64
	cpusum float64
}

// StartTheChallenge makes little children cry,
// don't worry though,
// i'm fly.
func StartTheChallenge() error {
	context := &Context{
		servers: make(map[string]*server),
	}
	log.Printf("Welcome to the CenturyLink Challenge, where all not-quite-snmp needs are fulfilled.")
	log.Printf("The time is: %v (this is RF3339)", time.Now().Format(time.RFC3339))
	router := mux.NewRouter()
	router.HandleFunc("/update", Handler{context, update}.ServeHTTP)
	router.HandleFunc("/get/{server}", Handler{context, get}.ServeHTTP)
	return http.ListenAndServe(":8080", router)
}

// The ServeHTTP abstraction that includes the current context
type Handler struct {
	*Context
	H func(*Context, http.ResponseWriter, *http.Request) (int, error)
}

// ServeHTTP function that utilizes contextual data
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, err := h.H(h.Context, w, r)
	if err != nil {
		switch status {
		case http.StatusNotFound:
			http.NotFound(w, r)
		case http.StatusInternalServerError:
			http.Error(w, http.StatusText(status), status)
		default:
			http.Error(w, http.StatusText(status), status)
		}
	}
}

// update receives an server pulse
func update(c *Context, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)
	stat := &pulse{}
	err := decoder.Decode(stat)
	if err != nil {
		log.Println(err)
		return 500, err
	}
	log.Printf("Received an update for server %v", stat.Name)
	return c.upsert(stat)
}

// get finds server information if available and returns an appropriate response
func get(c *Context, w http.ResponseWriter, r *http.Request) (int, error) {
	vars := mux.Vars(r)
	server := vars["server"]
	if _, exists := c.servers[server]; !exists {
		return 500, errors.New(fmt.Sprintf("No server information found for '%v'.", server))
	}
	if len(c.servers[server].Statistics) == 0 {
		fmt.Fprintf(w, "No update information for server %v found.", server)
		return 200, nil
	}
	mem60, cpu60 := c.servers[server].average(time.Duration(time.Minute * 60), time.Duration(time.Minute * 1))
	mem24, cpu24 := c.servers[server].average(time.Duration(time.Hour * 24), time.Duration(time.Hour * 1))

	fmt.Fprintf(w, fmt.Sprintf("Averages over the Last Hour: Memory: %v, CPU: %v. Last 24 Hours: Memory: %v, CPU: %v",
		mem60, cpu60, mem24, cpu24))
	log.Printf("Returning request for information for server %v.", server)
	return 200, nil
}

// upsert creates a statistic if needed and inserts the relevant data
func (c *Context) upsert(s *pulse) (int, error) {
	if _, exists := c.servers[s.Name]; !exists {
		c.servers[s.Name] = &server{
			Name: s.Name,
			Statistics: make([]*pulse, 0),
		}
	}
	c.servers[s.Name].Statistics = append(c.servers[s.Name].Statistics, s)
	return 200, nil
}

// average creates an average every duration specified by rate, up to length
func (s *server) average(length, rate time.Duration) ([]float64, []float64) {
	a := &average{
		memsum: float64(0.0),
		cpusum: float64(0.0),
		count: 0,
	}
	cpuaverages := make([]float64, 0) // create slices
	memaverages := make([]float64, 0) // create slices

	calculating := true
	start := time.Now()
	durations := 1
	for i := len(s.Statistics) - 1; calculating; i-- {
		// if the current pulse was sampled before the time range request, end calculating
		if i < 0 || s.Statistics[i].Time.Before(start.Add(-1 * length)) {
			if (a.cpusum > 0) {
				cpuaverages = append(cpuaverages, a.cpusum / float64(a.count))
				memaverages = append(memaverages, a.memsum / float64(a.count))
			}
			calculating = false
			continue
		}
		// if the current pulse is within the current duration, add it
		if s.Statistics[i].Time.After(start.Add(time.Duration(float64(-1.0 * durations) * rate.Minutes()) * time.Minute)) {
			a.count++
			a.cpusum += s.Statistics[i].Cpu
			a.memsum += s.Statistics[i].Mem
			continue
		}
		// if the current pulse is before the current duration, calculate current average and reset
		for s.Statistics[i].Time.Before(start.Add(time.Duration(float64(-1 * durations) * rate.Minutes()) * time.Minute)) {
			durations++
			cpuaverages, memaverages, a = appendAndReset(cpuaverages, memaverages, a)
		}
		// If this is the end of the pulses, clean up and move on
		if i == 0 {
			appendAndReset(cpuaverages, memaverages, a)
		}
	}
	return memaverages, cpuaverages
}

// appendAndReset adds the current statistics to the running list and resets the average
func appendAndReset(cpuaverages, memaverages []float64, a *average) ([]float64, []float64, *average) {
	if (a.cpusum > 0) {
		cpuaverages = append(cpuaverages, a.cpusum / float64(a.count))
		memaverages = append(memaverages, a.memsum / float64(a.count))
		a = &average{
			cpusum: float64(0),
			memsum: float64(0),
			count: 0,
		}
	}
	return cpuaverages, memaverages, a
}
