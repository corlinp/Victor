package main

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/corlinp/victor/vector"
	"github.com/dgraph-io/badger"
	"github.com/google/btree"
	"google.golang.org/protobuf/proto"
)

type VectorWithID struct {
	ID     string
	Vector *[1536]float64
}

type VectorIndex struct {
	tree *btree.BTreeG[VectorWithID]
	lock sync.RWMutex
}

func VectorWithIDLess(a, b VectorWithID) bool {
	return a.ID < b.ID
}

func NewVectorIndex(degree int) *VectorIndex {
	return &VectorIndex{
		tree: btree.NewG(degree, VectorWithIDLess),
	}
}

func (i *VectorIndex) Add(docID string, v *[1536]float64) {
	i.lock.Lock()
	defer i.lock.Unlock()
	data := VectorWithID{ID: docID, Vector: v}
	i.tree.ReplaceOrInsert(data)
}

func (i *VectorIndex) Delete(docID string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	data := VectorWithID{ID: docID}
	i.tree.Delete(data)
}

func (i *VectorIndex) Len() int {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.tree.Len()
}

func (i *VectorIndex) Search(v *[1536]float64, numResults int) ([]Result, error) {
	if numResults < 1 {
		return nil, fmt.Errorf("numResults must be greater than 0")
	}

	i.lock.RLock()
	defer i.lock.RUnlock()

	h := &ResultHeap{}
	heap.Init(h)

	i.tree.Ascend(func(item VectorWithID) bool {
		similarity := dotProduct(v, item.Vector)
		if h.Len() < numResults {
			heap.Push(h, Result{docID: item.ID, similarity: similarity})
		} else if similarity > h.Peek().similarity {
			heap.Pop(h)
			heap.Push(h, Result{docID: item.ID, similarity: similarity})
		}
		return true
	})

	results := make([]Result, h.Len())
	for i := range results {
		results[i] = heap.Pop(h).(Result)
	}

	return results, nil
}

type Result struct {
	docID      string
	similarity float64
}

type ResultHeap []Result

func (h ResultHeap) Len() int           { return len(h) }
func (h ResultHeap) Less(i, j int) bool { return h[i].similarity < h[j].similarity }
func (h ResultHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *ResultHeap) Push(x interface{}) {
	*h = append(*h, x.(Result))
}

func (h *ResultHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

func (h ResultHeap) Peek() Result {
	return h[0]
}

func dotProduct(a, b *[1536]float64) float64 {
	res := 0.0
	for i := 0; i < 1536; i++ {
		res += a[i] * b[i]
	}
	return res
}

func (index *VectorIndex) restoreIndex(db *badger.DB) {
	stream := db.NewStream()
	stream.NumGo = 16
	stream.LogPrefix = "BuildIndex"
	stream.Prefix = []byte(vectorPrefix)
	stream.Send = func(list *badger.KVList) error {
		for _, item := range list.Kv {
			var v vector.Vector
			err := proto.Unmarshal(item.Value, &v)
			if err != nil {
				return err
			}
			docID := string(item.Key)[2:]
			index.Add(docID, (*[1536]float64)(v.Values))
		}
		return nil
	}
	err := stream.Orchestrate(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
