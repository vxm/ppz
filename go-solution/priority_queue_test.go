package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestBasicPriorityQueue demonstrates basic usage
func TestBasicPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue()

	// Add items with different priorities
	pq.Add(10, 12345, "board_state_1")
	pq.Add(5, 23456, "board_state_2")
	pq.Add(15, 34567, "board_state_3")
	pq.Add(1, 45678, "board_state_4")

	// Should extract in priority order: 1, 5, 10, 15
	expected := []int{1, 5, 10, 15}

	for i, expectedPriority := range expected {
		item, ok := pq.TryPopMin()
		if !ok {
			t.Fatalf("Expected item %d, but queue was empty", i)
		}
		if item.Priority != expectedPriority {
			t.Errorf("Expected priority %d, got %d", expectedPriority, item.Priority)
		}
	}

	// Queue should be empty now
	if !pq.IsEmpty() {
		t.Error("Queue should be empty")
	}
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	pq := NewPriorityQueue()
	numWorkers := 4
	itemsPerWorker := 1000

	var wg sync.WaitGroup

	// Start producer goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < itemsPerWorker; j++ {
				priority := rand.Intn(10000)
				boardHash := uint64(workerID*1000000 + j)
				boardData := fmt.Sprintf("worker_%d_item_%d", workerID, j)
				pq.Add(priority, boardHash, boardData)
			}
		}(i)
	}

	// Wait for all producers to finish
	wg.Wait()

	// Now consume all items and verify order
	consumed := make([]QueueItem, 0, numWorkers*itemsPerWorker)

	for !pq.IsEmpty() {
		item, ok := pq.TryPopMin()
		if ok {
			consumed = append(consumed, item)
		}
	}

	// Verify all items were consumed and in correct order
	if len(consumed) != numWorkers*itemsPerWorker {
		t.Errorf("Expected %d items, got %d", numWorkers*itemsPerWorker, len(consumed))
	}

	// Verify order (each item should have priority >= previous)
	for i := 1; i < len(consumed); i++ {
		if consumed[i].Priority < consumed[i-1].Priority {
			t.Errorf("Items not in order: item %d has priority %d, previous had %d",
				i, consumed[i].Priority, consumed[i-1].Priority)
		}
	}
}

// DemonstratePriorityQueue shows usage example for Klotski solving
func DemonstratePriorityQueue() {
	fmt.Println("=== Priority Queue Demonstration ===")

	pq := NewPriorityQueue()

	// Simulate A* search for Klotski
	// Priority = g(cost so far) + h(heuristic to goal)

	// Initial state
	pq.Add(10, 17699058577533689103, "initial_board_state")

	// Add some possible next states with different priorities
	pq.Add(12, 1774575843912802194, "move_l_right_state")
	pq.Add(11, 8934751923847562812, "move_m_left_state")
	pq.Add(15, 4729384756291847563, "move_a_down_state")
	pq.Add(9, 2847592384756218394, "promising_state")

	fmt.Printf("Queue size: %d\n", pq.Size())

	// Process states in priority order (A* algorithm style)
	for !pq.IsEmpty() {
		item := pq.PopMin()
		fmt.Printf("Processing state with priority %d, hash %d\n",
			item.Priority, item.BoardHash)

		// In real A* implementation, you would:
		// 1. Check if this is the goal state
		// 2. Generate successor states
		// 3. Add promising successors to queue
	}

	fmt.Println("All states processed!")
	fmt.Println()
}

// BenchmarkStandardQueue measures performance of the standard priority queue
func BenchmarkStandardQueue(b *testing.B) {
	pq := NewPriorityQueue()

	b.ResetTimer()

	// Benchmark adding and removing items
	for i := 0; i < b.N; i++ {
		// Add some items
		pq.Add(rand.Intn(1000), uint64(i), fmt.Sprintf("state_%d", i))

		// Remove item if queue not empty
		if i%2 == 0 && !pq.IsEmpty() {
			pq.TryPopMin()
		}
	}
}

// BenchmarkTwoTierQueue measures performance of the two-tier queue
func BenchmarkTwoTierQueue(b *testing.B) {
	ttpq := NewTwoTierPriorityQueue(100) // Buffer threshold of 100

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Add some items
		ttpq.Add(rand.Intn(1000), uint64(i), fmt.Sprintf("state_%d", i))

		// Remove item occasionally
		if i%2 == 0 && i > 0 {
			ttpq.PopMin()
		}
	}
}

// BenchmarkConcurrentStandard tests concurrent performance of standard queue
func BenchmarkConcurrentStandard(b *testing.B) {
	pq := NewPriorityQueue()
	numGoroutines := runtime.NumCPU()

	b.ResetTimer()

	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < b.N/numGoroutines; i++ {
				// Mix of adds and removes
				if i%3 == 0 {
					pq.TryPopMin()
				} else {
					priority := rand.Intn(1000)
					pq.Add(priority, uint64(goroutineID*1000000+i), "state")
				}
			}
		}(g)
	}

	wg.Wait()
}

// ComparativeDemo shows performance characteristics of both approaches
func ComparativeDemo() {
	fmt.Println("=== Comparative Performance Demo ===")

	// Test with different workloads
	workloads := []struct {
		name    string
		adds    int
		removes int
	}{
		{"Light workload", 1000, 500},
		{"Heavy workload", 10000, 5000},
		{"Add-heavy", 10000, 1000},
	}

	for _, workload := range workloads {
		fmt.Printf("\n--- %s (%d adds, %d removes) ---\n",
			workload.name, workload.adds, workload.removes)

		// Test standard queue
		start := time.Now()
		pq := NewPriorityQueue()

		for i := 0; i < workload.adds; i++ {
			pq.Add(rand.Intn(10000), uint64(i), fmt.Sprintf("state_%d", i))
		}

		for i := 0; i < workload.removes && !pq.IsEmpty(); i++ {
			pq.TryPopMin()
		}

		standardTime := time.Since(start)
		fmt.Printf("Standard queue: %v\n", standardTime)

		// Test two-tier queue
		start = time.Now()
		ttpq := NewTwoTierPriorityQueue(100)

		for i := 0; i < workload.adds; i++ {
			ttpq.Add(rand.Intn(10000), uint64(i), fmt.Sprintf("state_%d", i))
		}

		for i := 0; i < workload.removes; i++ {
			ttpq.PopMin()
		}

		twoTierTime := time.Since(start)
		fmt.Printf("Two-tier queue: %v\n", twoTierTime)

		if standardTime < twoTierTime {
			fmt.Printf("Standard queue was %.2fx faster\n",
				float64(twoTierTime)/float64(standardTime))
		} else {
			fmt.Printf("Two-tier queue was %.2fx faster\n",
				float64(standardTime)/float64(twoTierTime))
		}
	}
}

// Main function to run demonstrations
func runQueueDemo() {
	fmt.Println("Thread-Safe Priority Queue Implementation for Klotski Solver")
	fmt.Println("===========================================================")
	fmt.Println()

	DemonstratePriorityQueue()
	ComparativeDemo()

	fmt.Println()
	fmt.Println("=== Memory Usage Comparison ===")
	fmt.Println("Standard Queue: ~24 bytes per item + heap overhead")
	fmt.Println("Two-Tier Queue: Similar memory usage but with buffering benefits")
	fmt.Println()
	fmt.Println("Recommendation: Use Standard Queue for most cases")
	fmt.Println("Use Two-Tier Queue only for extremely high-throughput scenarios")
}
