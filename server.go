package main

import (
	"encoding/json"
	"net/http"

	"github.com/corlinp/victor/vector"
	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
)

type Server struct {
	db    *badger.DB
	index *VectorIndex
}

func NewServer(db *badger.DB, index *VectorIndex) *Server {
	return &Server{
		db:    db,
		index: index,
	}
}

func (s *Server) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/add", s.AddHandler).Methods(http.MethodPut)
	r.HandleFunc("/search", s.SearchHandler).Methods(http.MethodPost)
	r.HandleFunc("/get/{id}", s.GetHandler).Methods(http.MethodGet)
	r.HandleFunc("/delete/{id}", s.DeleteHandler).Methods(http.MethodDelete)
}

const vectorPrefix = "v_"
const dataPrefix = "d_"

type VectorData struct {
	ID     string         `json:"id"`
	Vector *[1536]float64 `json:"vector"`
	Data   string         `json:"data"`
}

func (s *Server) AddHandler(w http.ResponseWriter, r *http.Request) {
	var vd VectorData
	err := json.NewDecoder(r.Body).Decode(&vd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		serr := txn.Set([]byte(dataPrefix+vd.ID), []byte(vd.Data))
		if serr != nil {
			return serr
		}
		vdProto := vector.Vector{
			Values: vd.Vector[:],
		}
		pdata, serr := proto.Marshal(&vdProto)
		if serr != nil {
			return serr
		}
		serr = txn.Set([]byte(vectorPrefix+vd.ID), pdata)
		if serr != nil {
			return serr
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.index.Add(vd.ID, vd.Vector)

	w.WriteHeader(http.StatusCreated)
}

type SearchRequest struct {
	Vector *[1536]float64 `json:"vector"`
	Count  int            `json:"count"`
}

type SearchResult struct {
	ID       string  `json:"id"`
	Data     string  `json:"data"`
	Distance float64 `json:"distance"`
}

func (s *Server) SearchHandler(w http.ResponseWriter, r *http.Request) {
	var sr SearchRequest
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results, err := s.index.Search(sr.Vector, sr.Count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var sresults []SearchResult
	for _, result := range results {
		var vd VectorData
		err := s.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(dataPrefix + result.docID))
			if err != nil {
				return err
			}

			data, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			vd.Data = string(data)
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sresults = append(sresults, SearchResult{
			ID:       result.docID,
			Data:     vd.Data,
			Distance: result.similarity,
		})
	}

	err = json.NewEncoder(w).Encode(sresults)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var outdata []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(dataPrefix + id))
		if err != nil {
			return err
		}
		outdata, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Write(outdata)
}

func (s *Server) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := s.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(dataPrefix + id))
		if err != nil {
			return err
		}
		err = txn.Delete([]byte(vectorPrefix + id))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
