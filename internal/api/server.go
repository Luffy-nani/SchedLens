package api

import (
	"SchedLens/internal/snapshot"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	db *snapshot.MongoDB
} // This is just a design pattern--> We store db in the server struct so that everytime we dont have to pass it to the handlers(if there are many handler func) and make the handler func method of this class **REMEMBER

func NewServer(db *snapshot.MongoDB) *Server {
	return &Server{db: db}
}

func (s *Server) Run(port string) {
	r := gin.Default() // Same as const app=express()

	r.GET("/health", s.health)
	r.GET("/processes", s.processes)
	r.GET("/history", s.history)
	r.GET("/starvation", s.starvation)

	r.Run(port) // Same as app.listen(port)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) processes(c *gin.Context) {
	to := time.Now()
	from := to.Add(-2 * time.Second) // we cannot just do to-2 because time.Time isnt a number

	data, err := s.db.Query(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(data) == 0 {
		c.JSON(http.StatusOK, gin.H{"processes": []interface{}{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"processes": data[len(data)-1].Processes}) // we're returning the latest data
}
func (s *Server) history(c *gin.Context) {
	pidStr := c.Query("pid")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	// Validate required query params
	if pidStr == "" || fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "pid, from, to are required",
		})
		return
	}

	// Parse PID
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid pid",
		})
		return
	}

	// Parse timestamps
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid from timestamp",
		})
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid to timestamp",
		})
		return
	}

	// Query snapshots in time range
	docs, err := s.db.Query(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Collect matching process history
	var history []snapshot.ProcessSnapshot

	for _, doc := range docs {
		for _, p := range doc.Processes {

			if p.PID == pid {
				history = append(history, p)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"pid":     pid,
		"history": history,
	})
}

func (s *Server) starvation(c *gin.Context) {
	to := time.Now()
	from := to.Add(-2 * time.Second)

	docs, err := s.db.Query(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(docs) == 0 {
		c.JSON(http.StatusOK, gin.H{"starved": []interface{}{}})
		return
	}

	// Filter starved processes from latest snapshot
	var starved []snapshot.ProcessSnapshot
	for _, p := range docs[len(docs)-1].Processes {
		if p.IsStarved {
			starved = append(starved, p)
		}
	}

	c.JSON(http.StatusOK, gin.H{"starved": starved})
}
