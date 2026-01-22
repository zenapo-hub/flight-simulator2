package api

import (
	"context"
	"encoding/json"
	"errors"
	"flight-simulator2/internal/sim"
	"fmt"
	"net/http"
	"time"
)

const (
	maxJSONBodyBytes = 1 << 20 // 1MB
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
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) state(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	st, err := s.eng.GetState(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}
	writeJSON(w, http.StatusOK, st)
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

	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate inputs
	if err := validateLatLon(body.Lat, body.Lon); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Alt < -500 {
		jsonError(w, http.StatusBadRequest, "alt must be >= -500 meters")
		return
	}
	if body.Speed < 0 {
		jsonError(w, http.StatusBadRequest, "speed must be >= 0")
		return
	}

	s.eng.Submit(sim.GoToCommand{
		At:    time.Now(),
		Lat:   body.Lat,
		Lon:   body.Lon,
		Alt:   body.Alt,
		Speed: body.Speed,
	})

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted", "type": "goto"})
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

	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(body.Waypoints) == 0 {
		jsonError(w, http.StatusBadRequest, "waypoints required")
		return
	}

	// Validate each waypoint
	for i, wp := range body.Waypoints {
		if err := validateLatLon(wp.Lat, wp.Lon); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Sprintf("waypoints[%d]: %s", i, err.Error()))
			return
		}
		if wp.Alt < -500 {
			jsonError(w, http.StatusBadRequest, fmt.Sprintf("waypoints[%d]: alt must be >= -500 meters", i))
			return
		}
		if wp.Speed < 0 {
			jsonError(w, http.StatusBadRequest, fmt.Sprintf("waypoints[%d]: speed must be >= 0", i))
			return
		}
	}

	s.eng.Submit(sim.TrajectoryCommand{
		At:        time.Now(),
		Waypoints: body.Waypoints,
		Loop:      body.Loop,
	})

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": "accepted",
		"type":   "trajectory",
		"count":  len(body.Waypoints),
	})
}

func (s *Server) stopCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	s.eng.Submit(sim.StopCommand{At: time.Now()})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted", "type": "stop"})
}

func (s *Server) holdCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	s.eng.Submit(sim.HoldCommand{At: time.Now()})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted", "type": "hold"})
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

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Helps with Nginx / reverse-proxy buffering
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()
	ch, unsub := s.eng.Subscribe(ctx)
	defer unsub()

	// comment line (keeps some proxies happy)
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
			b, err := json.Marshal(st)
			if err != nil {
				// if marshal fails, end stream (rare)
				return
			}
			fmt.Fprintf(w, "event: state\n")
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

// ---- helpers ----

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return fmt.Errorf("invalid json syntax at byte %d", syntaxErr.Offset)
		}
		return fmt.Errorf("invalid json: %w", err)
	}
	// Ensure there's no extra trailing content
	if dec.More() {
		return fmt.Errorf("invalid json: multiple values in body")
	}
	return nil
}

func validateLatLon(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("lat must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return fmt.Errorf("lon must be between -180 and 180")
	}
	return nil
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{
		"error":  msg,
		"status": "rejected",
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
