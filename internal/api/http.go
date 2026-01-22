package api

import (
	"context"
	"encoding/json"
	"flight-simulator2/internal/sim"
	"fmt"
	"net/http"
	"time"
)

type Server struct {
	eng *sim.Engine
	mux *http.ServeMux
}

func NewServer(eng *sim.Engine) *Server {
	s := &Server{eng: eng, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.health)
	s.mux.HandleFunc("/state", s.state)

	s.mux.HandleFunc("/command/goto", s.gotoCmd)
	s.mux.HandleFunc("/command/trajectory", s.trajectoryCmd)

	s.mux.HandleFunc("/command/stop", s.stopCmd)
	s.mux.HandleFunc("/command/hold", s.holdCmd)

	s.mux.HandleFunc("/stream", s.streamSSE)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) state(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	st, err := s.eng.GetState(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}
	writeJSON(w, st)
}

func (s *Server) gotoCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Lat   float64 `json:"lat"`
		Lon   float64 `json:"lon"`
		Alt   float64 `json:"alt"`
		Speed float64 `json:"speed,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	s.eng.Submit(sim.GoToCommand{
		At:    time.Now(),
		Lat:   body.Lat,
		Lon:   body.Lon,
		Alt:   body.Alt,
		Speed: body.Speed,
	})

	writeJSON(w, map[string]any{"status": "accepted", "type": "goto"})
}

func (s *Server) trajectoryCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Waypoints []sim.Waypoint `json:"waypoints"`
		Loop      bool           `json:"loop,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(body.Waypoints) == 0 {
		http.Error(w, "waypoints required", http.StatusBadRequest)
		return
	}

	s.eng.Submit(sim.TrajectoryCommand{
		At:        time.Now(),
		Waypoints: body.Waypoints,
		Loop:      body.Loop,
	})

	writeJSON(w, map[string]any{"status": "accepted", "type": "trajectory", "count": len(body.Waypoints)})
}

func (s *Server) stopCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	s.eng.Submit(sim.StopCommand{At: time.Now()})
	writeJSON(w, map[string]any{"status": "accepted", "type": "stop"})
}

func (s *Server) holdCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	s.eng.Submit(sim.HoldCommand{At: time.Now()})
	writeJSON(w, map[string]any{"status": "accepted", "type": "hold"})
}

func (s *Server) streamSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	ch, unsub := s.eng.Subscribe(ctx)
	defer unsub()

	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case st, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(st)
			fmt.Fprintf(w, "event: state\n")
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
