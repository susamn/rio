package rio

import (
	"container/heap"
	"fmt"
	"testing"
)

func TestPool(t *testing.T) {
	// Some items and their priorities.
	requests := []*Request{
		{}, {}, {}, {}, {},
	}

	// Create a priority queue, put the items in it, and
	// establish the priority queue (heap) invariants.
	poo := make(Pool, len(requests))
	i := 0
	for p, _ := range requests {
		w := &Worker{
			pending: p,
			index:   i,
		}
		poo[i] = w
		i++
	}
	for _, v := range poo {
		fmt.Println(v.pending)
	}

	for i, _ := range poo {
		if i == 2 {
			poo[i].pending = 5
		}

		heap.Fix(&poo, i)
	}
	item := heap.Pop(&poo).(*Worker)

	if item.pending != 0 {
		t.Fail()
	}
}
