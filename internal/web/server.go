package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/Blustak/go-factorioHelper/internal/catalog"
	"github.com/Blustak/go-factorioHelper/internal/chain"
	"github.com/Blustak/go-factorioHelper/internal/config"
)

type Server struct {
	cfg     *config.State
	catalog *catalog.Catalog
	mux     *http.ServeMux
}

func New(cfg *config.State) (*Server, error) {
	cat, err := catalog.Load(cfg)
	if err != nil {
		return nil, err
	}
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:     cfg,
		catalog: cat,
		mux:     http.NewServeMux(),
	}
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("GET /api/recipes", s.handleRecipes)
	s.mux.HandleFunc("GET /api/items", s.handleItems)
	s.mux.HandleFunc("GET /api/fluids", s.handleFluids)
	s.mux.HandleFunc("GET /api/machines", s.handleMachines)
	s.mux.HandleFunc("GET /api/producers", s.handleProducers)
	s.mux.HandleFunc("GET /api/boilers", s.handleBoilers)
	s.mux.HandleFunc("GET /api/generators", s.handleGenerators)
	s.mux.HandleFunc("POST /api/graph/validate", s.handleValidate)
	s.mux.HandleFunc("POST /api/graph/analyze", s.handleAnalyze)
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleRecipes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.catalog.Recipes)
}

func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.catalog.Items)
}

func (s *Server) handleFluids(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.catalog.Fluids)
}

func (s *Server) handleMachines(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	writeJSON(w, s.catalog.MachinesForCategory(category))
}

func (s *Server) handleProducers(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	writeJSON(w, s.catalog.ProducersForCategory(category))
}

func (s *Server) handleBoilers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.catalog.Boilers)
}

func (s *Server) handleGenerators(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.catalog.Generators)
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	g, err := decodeGraph(r)
	if err != nil {
		http.Error(w, "invalid graph JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, chain.Validate(g, s.catalog))
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	g, err := decodeGraph(r)
	if err != nil {
		http.Error(w, "invalid graph JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, chain.Analyze(g, s.catalog))
}

func decodeGraph(r *http.Request) (chain.Graph, error) {
	defer r.Body.Close()
	var g chain.Graph
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		return g, err
	}
	if g.Nodes == nil {
		g.Nodes = []chain.NodeDoc{}
	}
	if g.Edges == nil {
		g.Edges = []chain.Edge{}
	}
	return g, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
