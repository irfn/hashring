package hashring

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// Param: expect functions will not work. They assume the old hash function.
func expectNode(t *testing.T, hashRing *HashRing, key string, expectedNode string) {
	node, ok := hashRing.GetNode(key)
	if !ok || node != expectedNode {
		t.Error("GetNode(", key, ") expected", expectedNode, "but got", node)
	}
}

func expectNodes(t *testing.T, hashRing *HashRing, key string, expectedNodes []string) {
	nodes, ok := hashRing.GetNodes(key, 2)
	sliceEquality := reflect.DeepEqual(nodes, expectedNodes)
	if !ok || !sliceEquality {
		t.Error("GetNodes(", key, ") expected", expectedNodes, "but got", nodes)
	}
}

func expectNodesABC(t *testing.T, hashRing *HashRing) {
	// Python hash_ring module test case
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "c")
	expectNode(t, hashRing, "test5", "a")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "a")
}

func expectNodeRangesABC(t *testing.T, hashRing *HashRing) {
	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test1", []string{"b", "c"})
	expectNodes(t, hashRing, "test2", []string{"b", "a"})
	expectNodes(t, hashRing, "test3", []string{"c", "a"})
	expectNodes(t, hashRing, "test4", []string{"c", "b"})
	expectNodes(t, hashRing, "test5", []string{"a", "c"})
	expectNodes(t, hashRing, "aaaa", []string{"b", "a"})
	expectNodes(t, hashRing, "bbbb", []string{"a", "b"})
}

func expectNodesABCD(t *testing.T, hashRing *HashRing) {
	// Somehow adding d does not load balance these keys...
	expectNodesABC(t, hashRing)
}

func TestNew(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	hashRing := New(nodes)

	expectNodesABC(t, hashRing)
	expectNodeRangesABC(t, hashRing)
}

func TestNewEmpty(t *testing.T) {
	nodes := []string{}
	hashRing := New(nodes)

	node, ok := hashRing.GetNode("test")
	if ok || node != "" {
		t.Error("GetNode(test) expected (\"\", false) but got (", node, ",", ok, ")")
	}

	nodes, rok := hashRing.GetNodes("test", 2)
	if rok || !(len(nodes) == 0) {
		t.Error("GetNode(test) expected ( [], false ) but got (", nodes, ",", rok, ")")
	}
}

func TestForMoreNodes(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	hashRing := New(nodes)

	nodes, ok := hashRing.GetNodes("test", 5)
	if ok || !(len(nodes) == 0) {
		t.Error("GetNode(test) expected ( [], false ) but got (", nodes, ",", ok, ")")
	}
}

func TestForEqualNodes(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	hashRing := New(nodes)

	nodes, ok := hashRing.GetNodes("test", 3)
	if !ok && (len(nodes) == 3) {
		t.Error("GetNode(test) expected ( [a b c], true ) but got (", nodes, ",", ok, ")")
	}
}

func TestNewSingle(t *testing.T) {
	nodes := []string{"a"}
	hashRing := New(nodes)

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "a")
	expectNode(t, hashRing, "test2", "a")
	expectNode(t, hashRing, "test3", "a")

	// This triggers the edge case where sortedKey search resulting in not found
	expectNode(t, hashRing, "test14", "a")

	expectNode(t, hashRing, "test15", "a")
	expectNode(t, hashRing, "test16", "a")
	expectNode(t, hashRing, "test17", "a")
	expectNode(t, hashRing, "test18", "a")
	expectNode(t, hashRing, "test19", "a")
	expectNode(t, hashRing, "test20", "a")
}

func TestNewWeighted(t *testing.T) {
	weights := make(map[string]int)
	weights["a"] = 1
	weights["b"] = 2
	weights["c"] = 1
	hashRing := NewWithWeights(weights)

	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "b")
	expectNode(t, hashRing, "test5", "b")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "a")

	expectNodes(t, hashRing, "test", []string{"b", "a"})
}

func TestRemoveNode(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	hashRing := New(nodes)
	hashRing = hashRing.RemoveNode("b")

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "c") // Migrated to c from b
	expectNode(t, hashRing, "test2", "a") // Migrated to a from b
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "c")
	expectNode(t, hashRing, "test5", "a")
	expectNode(t, hashRing, "aaaa", "a") // Migrated to a from b
	expectNode(t, hashRing, "bbbb", "a")

	expectNodes(t, hashRing, "test", []string{"a", "c"})
}

func TestAddNode(t *testing.T) {
	nodes := []string{"a", "c"}
	hashRing := New(nodes)
	hashRing = hashRing.AddNode("b")

	expectNodesABC(t, hashRing)
}

func TestAddNode2(t *testing.T) {
	nodes := []string{"a", "c"}
	hashRing := New(nodes)
	hashRing = hashRing.AddNode("b")
	hashRing = hashRing.AddNode("b")

	expectNodesABC(t, hashRing)
	expectNodeRangesABC(t, hashRing)
}

func TestAddNode3(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	hashRing := New(nodes)
	hashRing = hashRing.AddNode("d")

	// Somehow adding d does not load balance these keys...
	expectNodesABCD(t, hashRing)

	hashRing = hashRing.AddNode("e")

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "c")
	expectNode(t, hashRing, "test5", "a")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "e") // Migrated to e from a

	expectNodes(t, hashRing, "test", []string{"a", "b"})

	hashRing = hashRing.AddNode("f")

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "f") // Migrated to f from b
	expectNode(t, hashRing, "test3", "f") // Migrated to f from c
	expectNode(t, hashRing, "test4", "c")
	expectNode(t, hashRing, "test5", "f") // Migrated to f from a
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "e")

	expectNodes(t, hashRing, "test", []string{"a", "b"})
}

func TestDuplicateNodes(t *testing.T) {
	nodes := []string{"a", "a", "a", "a", "b"}
	hashRing := New(nodes)

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "a")
	expectNode(t, hashRing, "test4", "b")
	expectNode(t, hashRing, "test5", "a")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "a")
}

func TestAddWeightedNode(t *testing.T) {
	nodes := []string{"a", "c"}
	hashRing := New(nodes)
	hashRing = hashRing.AddWeightedNode("b", 0)
	hashRing = hashRing.AddWeightedNode("b", 2)
	hashRing = hashRing.AddWeightedNode("b", 2)

	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "b")
	expectNode(t, hashRing, "test5", "b")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "a")

	expectNodes(t, hashRing, "test", []string{"b", "a"})
}

func TestUpdateWeightedNode(t *testing.T) {
	nodes := []string{"a", "c"}
	hashRing := New(nodes)
	hashRing = hashRing.AddWeightedNode("b", 1)
	hashRing = hashRing.UpdateWeightedNode("b", 2)
	hashRing = hashRing.UpdateWeightedNode("b", 2)
	hashRing = hashRing.UpdateWeightedNode("b", 0)
	hashRing = hashRing.UpdateWeightedNode("d", 2)

	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "b")
	expectNode(t, hashRing, "test5", "b")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "a")

	expectNodes(t, hashRing, "test", []string{"b", "a"})
}

func TestRemoveAddNode(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	hashRing := New(nodes)

	expectNodesABC(t, hashRing)
	expectNodeRangesABC(t, hashRing)

	hashRing = hashRing.RemoveNode("b")

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "c") // Migrated to c from b
	expectNode(t, hashRing, "test2", "a") // Migrated to a from b
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "c")
	expectNode(t, hashRing, "test5", "a")
	expectNode(t, hashRing, "aaaa", "a") // Migrated to a from b
	expectNode(t, hashRing, "bbbb", "a")

	expectNodes(t, hashRing, "test", []string{"a", "c"})
	expectNodes(t, hashRing, "test", []string{"a", "c"})
	expectNodes(t, hashRing, "test1", []string{"c", "a"})
	expectNodes(t, hashRing, "test2", []string{"a", "c"})
	expectNodes(t, hashRing, "test3", []string{"c", "a"})
	expectNodes(t, hashRing, "test4", []string{"c", "a"})
	expectNodes(t, hashRing, "test5", []string{"a", "c"})
	expectNodes(t, hashRing, "aaaa", []string{"a", "c"})
	expectNodes(t, hashRing, "bbbb", []string{"a", "c"})

	hashRing = hashRing.AddNode("b")

	expectNodesABC(t, hashRing)
	expectNodeRangesABC(t, hashRing)
}

func TestRemoveAddWeightedNode(t *testing.T) {
	weights := make(map[string]int)
	weights["a"] = 1
	weights["b"] = 2
	weights["c"] = 1
	hashRing := NewWithWeights(weights)

	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "b")
	expectNode(t, hashRing, "test5", "b")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "a")

	expectNodes(t, hashRing, "test", []string{"b", "a"})
	expectNodes(t, hashRing, "test", []string{"b", "a"})
	expectNodes(t, hashRing, "test1", []string{"b", "c"})
	expectNodes(t, hashRing, "test2", []string{"b", "a"})
	expectNodes(t, hashRing, "test3", []string{"c", "b"})
	expectNodes(t, hashRing, "test4", []string{"b", "a"})
	expectNodes(t, hashRing, "test5", []string{"b", "a"})
	expectNodes(t, hashRing, "aaaa", []string{"b", "a"})
	expectNodes(t, hashRing, "bbbb", []string{"a", "b"})

	hashRing = hashRing.RemoveNode("c")

	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test", "b")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "b") // Migrated to b from c
	expectNode(t, hashRing, "test4", "b")
	expectNode(t, hashRing, "test5", "b")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "a")

	expectNodes(t, hashRing, "test", []string{"b", "a"})
	expectNodes(t, hashRing, "test", []string{"b", "a"})
	expectNodes(t, hashRing, "test1", []string{"b", "a"})
	expectNodes(t, hashRing, "test2", []string{"b", "a"})
	expectNodes(t, hashRing, "test3", []string{"b", "a"})
	expectNodes(t, hashRing, "test4", []string{"b", "a"})
	expectNodes(t, hashRing, "test5", []string{"b", "a"})
	expectNodes(t, hashRing, "aaaa", []string{"b", "a"})
	expectNodes(t, hashRing, "bbbb", []string{"a", "b"})
}

func TestAddRemoveNode(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	hashRing := New(nodes)
	hashRing = hashRing.AddNode("d")

	// Somehow adding d does not load balance these keys...
	expectNodesABCD(t, hashRing)

	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test1", []string{"b", "d"})
	expectNodes(t, hashRing, "test2", []string{"b", "d"})
	expectNodes(t, hashRing, "test3", []string{"c", "d"})
	expectNodes(t, hashRing, "test4", []string{"c", "b"})
	expectNodes(t, hashRing, "test5", []string{"a", "d"})
	expectNodes(t, hashRing, "aaaa", []string{"b", "a"})
	expectNodes(t, hashRing, "bbbb", []string{"a", "b"})

	hashRing = hashRing.AddNode("e")

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "b")
	expectNode(t, hashRing, "test3", "c")
	expectNode(t, hashRing, "test4", "c")
	expectNode(t, hashRing, "test5", "a")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "e") // Migrated to e from a

	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test1", []string{"b", "d"})
	expectNodes(t, hashRing, "test2", []string{"b", "d"})
	expectNodes(t, hashRing, "test3", []string{"c", "e"})
	expectNodes(t, hashRing, "test4", []string{"c", "b"})
	expectNodes(t, hashRing, "test5", []string{"a", "e"})
	expectNodes(t, hashRing, "aaaa", []string{"b", "e"})
	expectNodes(t, hashRing, "bbbb", []string{"e", "a"})

	hashRing = hashRing.AddNode("f")

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "f") // Migrated to f from b
	expectNode(t, hashRing, "test3", "f") // Migrated to f from c
	expectNode(t, hashRing, "test4", "c")
	expectNode(t, hashRing, "test5", "f") // Migrated to f from a
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "e")

	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test1", []string{"b", "d"})
	expectNodes(t, hashRing, "test2", []string{"f", "b"})
	expectNodes(t, hashRing, "test3", []string{"f", "c"})
	expectNodes(t, hashRing, "test4", []string{"c", "b"})
	expectNodes(t, hashRing, "test5", []string{"f", "a"})
	expectNodes(t, hashRing, "aaaa", []string{"b", "e"})
	expectNodes(t, hashRing, "bbbb", []string{"e", "f"})

	hashRing = hashRing.RemoveNode("e")

	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test", "a")
	expectNode(t, hashRing, "test1", "b")
	expectNode(t, hashRing, "test2", "f")
	expectNode(t, hashRing, "test3", "f")
	expectNode(t, hashRing, "test4", "c")
	expectNode(t, hashRing, "test5", "f")
	expectNode(t, hashRing, "aaaa", "b")
	expectNode(t, hashRing, "bbbb", "f") // Migrated to f from e

	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test1", []string{"b", "d"})
	expectNodes(t, hashRing, "test2", []string{"f", "b"})
	expectNodes(t, hashRing, "test3", []string{"f", "c"})
	expectNodes(t, hashRing, "test4", []string{"c", "b"})
	expectNodes(t, hashRing, "test5", []string{"f", "a"})
	expectNodes(t, hashRing, "aaaa", []string{"b", "a"})
	expectNodes(t, hashRing, "bbbb", []string{"f", "a"})

	hashRing = hashRing.RemoveNode("f")

	expectNodesABCD(t, hashRing)

	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test", []string{"a", "b"})
	expectNodes(t, hashRing, "test1", []string{"b", "d"})
	expectNodes(t, hashRing, "test2", []string{"b", "d"})
	expectNodes(t, hashRing, "test3", []string{"c", "d"})
	expectNodes(t, hashRing, "test4", []string{"c", "b"})
	expectNodes(t, hashRing, "test5", []string{"a", "d"})
	expectNodes(t, hashRing, "aaaa", []string{"b", "a"})
	expectNodes(t, hashRing, "bbbb", []string{"a", "b"})

	hashRing = hashRing.RemoveNode("d")

	expectNodesABC(t, hashRing)
	expectNodeRangesABC(t, hashRing)
}

func BenchmarkHashes(b *testing.B) {
	nodes := []string{"a", "b", "c", "d", "e", "f", "g"}
	hashRing := New(nodes)
	tt := []struct {
		key   string
		nodes []string
	}{
		{"test", []string{"a", "b"}},
		{"test", []string{"a", "b"}},
		{"test1", []string{"b", "d"}},
		{"test2", []string{"f", "b"}},
		{"test3", []string{"f", "c"}},
		{"test4", []string{"c", "b"}},
		{"test5", []string{"f", "a"}},
		{"aaaa", []string{"b", "a"}},
		{"bbbb", []string{"f", "a"}},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		o := tt[i%len(tt)]
		hashRing.GetNodes(o.key, 2)
	}
}

func BenchmarkHashesSingle(b *testing.B) {
	nodes := []string{"a", "b", "c", "d", "e", "f", "g"}
	hashRing := New(nodes)
	tt := []struct {
		key   string
		nodes []string
	}{
		{"test", []string{"a", "b"}},
		{"test", []string{"a", "b"}},
		{"test1", []string{"b", "d"}},
		{"test2", []string{"f", "b"}},
		{"test3", []string{"f", "c"}},
		{"test4", []string{"c", "b"}},
		{"test5", []string{"f", "a"}},
		{"aaaa", []string{"b", "a"}},
		{"bbbb", []string{"f", "a"}},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		o := tt[i%len(tt)]
		hashRing.GetNode(o.key)
	}
}

// Test Weights distribution
// Test Hashing Distribution on various datatypes ID Location
// Performance test
func TestSimpleHashRing(t *testing.T) {
	serversInRing := []string{"192.168.0.246:11212",
		"192.168.0.247:11212",
		"192.168.0.248:11212",
		"192.168.0.249:11212",
		"192.168.0.250:11212",
		"192.168.0.251:11212",
		"192.168.0.252:11212"}

	ring := New(serversInRing)

	var results []string

	times := 1000000
	for i := 0; i < times; i++ {
		server, _ := ring.GetNode(string(i))
		results = append(results, server)
	}
	for i := 0; i < times; i++ {
		server, _ := ring.GetNode(string(i))
		if server != results[i] {
			t.Fatalf("Failure in comparison %s, %s", server, results[i])
		}
	}

}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func TestSimpleHashRingCompareMultipleRuns(t *testing.T) {
	serversInRing := []string{"192.168.0.246:11212",
		"192.168.0.247:11212",
		"192.168.0.248:11212",
		"192.168.0.249:11212",
		"192.168.0.250:11212",
		"192.168.0.251:11212",
		"192.168.0.252:11212"}

	ring := New(serversInRing)

	var buffer bytes.Buffer

	times := 10000
	for i := 0; i < times; i++ {
		server, _ := ring.GetNode(string(i))
		buffer.WriteString(fmt.Sprintf("%s\n", server))
	}
	bytes1 := buffer.Bytes()
	ioutil.WriteFile("/tmp/dat1", bytes1, 0644)

	var nBuffer bytes.Buffer
	for i := 0; i < times; i++ {
		server, _ := ring.GetNode(string(i))
		nBuffer.WriteString(fmt.Sprintf("%s\n", server))
	}
	bytes2 := nBuffer.Bytes()
	ioutil.WriteFile("/tmp/dat2", bytes2, 0644)

	if !bytes.Equal(bytes1, bytes2) {
		t.Fatal("Failure in comparison of bytes")
	}
}

func TestSimpleHashRingTiming(t *testing.T) {
	serversInRing := []string{"192.168.0.246:11212",
		"192.168.0.247:11212",
		"192.168.0.248:11212",
		"192.168.0.249:11212",
		"192.168.0.250:11212",
		"192.168.0.251:11212",
		"192.168.0.252:11212"}

	ring := New(serversInRing)

	times := 10000000
	start := time.Now()
	for i := 0; i < times; i++ {
		ring.GetNode(strconv.Itoa(i))
	}
	elapsed := time.Since(start)
	startSecond := time.Now()
	for i := 0; i < times; i++ {
		ring.GetNode(strconv.Itoa(i))
	}
	elapsedSecond := time.Since(startSecond)

	fmt.Printf("first elapsed %v, second elapsed %v", elapsed, elapsedSecond)
}

func TestSimpleHashRingMemoryUsage(t *testing.T) {
	serversInRing := []string{"192.168.0.246:11212",
		"192.168.0.247:11212",
		"192.168.0.248:11212",
		"192.168.0.249:11212",
		"192.168.0.250:11212",
		"192.168.0.251:11212",
		"192.168.0.252:11212"}

	ring := New(serversInRing)

	times := 100000000
	PrintMemUsage()
	for i := 0; i < times; i++ {
		ring.GetNode(string(i))
	}
	PrintMemUsage()
	for i := 0; i < times; i++ {
		ring.GetNode(string(i))
	}
	PrintMemUsage()
}

func TestShouldHandleWeights(t *testing.T) {
	ring := NewWithWeights(map[string]int{"a": 1}).
		AddWeightedNode("b", 1).
		AddWeightedNode("c", 1).
		AddWeightedNode("d", 1)
	times := 20
	var results []string
	for i := 0; i < times; i++ {
		server, _ := ring.GetNode(string(i))
		pos, _ := ring.GetNodePos(string(i))
		results = append(results, fmt.Sprintf("%d : %s : %d", i, server, pos))
	}
	ring = ring.RemoveNode("d").AddWeightedNode("d1", 1).AddWeightedNode("d2", 1)
	var buffer bytes.Buffer
	for i := 0; i < times; i++ {
		server, _ := ring.GetNode(string(i))
		pos, _ := ring.GetNodePos(string(i))
		buffer.WriteString(fmt.Sprintf("%s - %s - %d\n", results[i], server, pos))
	}

	fmt.Println(buffer.String())
}

func TestShouldHandleWeights2(t *testing.T) {
	ring := New([]string{
		"1",
		"2",
		"3",
		"4",
		"5",
		"6",
		"7",
		"8",
		"9",
		"10",
	})

	//	a: [0-500],
	//	b: [500-749],
	//	c: [750-100]
	serverMap := map[string]string{
		"1":  "a",
		"2":  "a",
		"3":  "a",
		"4":  "a",
		"5":  "a",
		"6":  "b",
		"7":  "b",
		"8":  "b",
		"9":  "c",
		"10": "c",
	}

	times := 1000
	serverData := map[string]int{"a": 0, "b": 0, "c": 0}
	var results []string
	for i := 0; i < times; i++ {
		server, _ := ring.GetNode(strconv.Itoa(i))
		results = append(results, fmt.Sprintf("%d : %s", i, serverMap[server]))
		serverData[serverMap[server]] = serverData[serverMap[server]] + 1
	}

	var buffer bytes.Buffer
	for i := 0; i < times; i++ {
		server, _ := ring.GetNode(strconv.Itoa(i))
		buffer.WriteString(fmt.Sprintf("%s - %s\n", results[i], serverMap[server]))
	}

	fmt.Println(serverData)
	fmt.Println(buffer.String())
}

func TestSimpleDistribution(t *testing.T) {
	serversInRing := []string{
		"a",
		"b",
		"c",
		"d",
		"e",
		"f",
		"g",
		"h",
		"i",
		"j",
	}

	ring := New(serversInRing)

	times := 10000000

	totals := map[string]int{"a": 0,
		"b": 0,
		"c": 0,
		"d": 0,
		"e": 0,
		"f": 0,
		"g": 0,
		"h": 0,
		"i": 0,
		"j": 0,
	}

	for i := 0; i < times; i++ {
		result, _ := ring.GetNode(strconv.Itoa(i))
		totals[result] = totals[result] + 1
	}

	t.Logf("%v", totals)
}
