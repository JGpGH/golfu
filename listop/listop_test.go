package listop_test

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/JGpGH/golfu/listop"
)

type testStruct struct {
	ID           string
	is_something bool
}

func (t *testStruct) Index() string {
	return t.ID
}

func Test_IndexedList_SetThenGet(t *testing.T) {
	indexedList := listop.NewIndexedList[*testStruct]()
	test1 := &testStruct{ID: "1", is_something: true}
	test2 := &testStruct{ID: "2", is_something: false}
	test3 := &testStruct{ID: "3", is_something: true}
	test4 := &testStruct{ID: "4", is_something: false}

	indexedList.Set([]*testStruct{test1, test2, test3, test4})
	r := indexedList.Gets([]string{"1", "2", "3", "4"})
	if len(r) != 4 {
		t.Error("Expected 4 elements, got ", len(r))
	}
	if r["1"] != test1 {
		t.Error("Expected test1, got ", r["1"])
	}
	if r["2"] != test2 {
		t.Error("Expected test2, got ", r["2"])
	}
	if r["3"] != test3 {
		t.Error("Expected test3, got ", r["3"])
	}
	if r["4"] != test4 {
		t.Error("Expected test4, got ", r["4"])
	}
	if r["4"].is_something {
		t.Error("Expected test4 to be false, got true")
	}
	indexedList.Set([]*testStruct{{ID: "4", is_something: true}})
	r = indexedList.Gets([]string{"4"})
	if r["4"].is_something == false {
		t.Error("Expected test4 to be true, got false")
	}
}

func Test_IndexedList_GetReadWriteCounts(t *testing.T) {
	indexedList := listop.NewIndexedList[*testStruct]()
	test1 := &testStruct{ID: "1", is_something: true}
	test2 := &testStruct{ID: "2", is_something: false}
	test3 := &testStruct{ID: "3", is_something: true}
	test4 := &testStruct{ID: "4", is_something: false}

	indexedList.Set([]*testStruct{test1, test2, test3, test4})
	indexedList.Gets([]string{"1", "2", "3", "4"})
	indexedList.Gets([]string{"1", "2", "3", "4"})
	indexedList.Gets([]string{"2"})
	expected := map[string]uint32{
		"1": 3,
		"2": 4,
		"3": 3,
		"4": 3,
	}
	r := indexedList.ReadWriteCounts([]string{"1", "2", "3", "4"})
	for k, v := range r {
		if v != expected[k] {
			t.Error("Expected ", expected[k], " for ID:", k, ", got ", v)
		}
	}

	indexedList.Gets([]string{"2"})
	expected["2"] = 5
	r = indexedList.ReadWriteCounts([]string{"1", "2", "3", "4"})
	for k, v := range r {
		if v != expected[k] {
			t.Error("Expected ", expected[k], " for ID:", k, ", got ", v)
		}
	}

	indexedList.Gets([]string{"3", "4"})
	indexedList.Gets([]string{"3", "4"})
	indexedList.Gets([]string{"3", "4"})

	expected["3"] = 6
	expected["4"] = 6

	r = indexedList.ReadWriteCounts([]string{"1", "2", "3", "4"})
	for k, v := range r {
		if v != expected[k] {
			t.Error("Expected ", expected[k], " for ID:", k, ", got ", v)
		}
	}
}

func Test_IndexedList_Remove(t *testing.T) {
	indexedList := listop.NewIndexedList[*testStruct]()
	test1 := &testStruct{ID: "1", is_something: true}
	test2 := &testStruct{ID: "2", is_something: false}
	test3 := &testStruct{ID: "3", is_something: true}
	test4 := &testStruct{ID: "4", is_something: false}

	indexedList.Set([]*testStruct{test1, test2, test3, test4})
	indexedList.Remove([]string{"2", "3"})

	r := indexedList.Gets([]string{"1", "2", "3", "4"})
	if len(r) != 2 {
		t.Error("Expected 2 elements, got ", len(r))
	}
	if r["1"] != test1 {
		t.Error("Expected test1, got ", r["1"])
	}
	if r["4"] != test4 {
		t.Error("Expected test4, got ", r["4"])
	}
	indexedList.Remove([]string{"1"})
	r = indexedList.Gets([]string{"1", "2", "3", "4"})
	if len(r) != 1 {
		t.Error("Expected 1 elements, got ", len(r))
	}
	if r["4"] != test4 {
		t.Error("Expected test4, got ", r["4"])
	}
}

func Test_IndexedList_PopWhere(t *testing.T) {
	indexedList := listop.NewIndexedList[*testStruct]()
	indexedList.Set([]*testStruct{
		{ID: "1", is_something: true},
		{ID: "2", is_something: false},
		{ID: "3", is_something: true},
		{ID: "4", is_something: false},
		{ID: "5", is_something: true},
		{ID: "6", is_something: false},
	})
	indexedList.PopWhere(func(t *testStruct) bool {
		return !t.is_something
	}, 2)
	r := indexedList.Gets([]string{"1", "2", "3", "4", "5", "6"})
	if len(r) != 4 {
		t.Error("Expected 4 elements, got ", len(r))
	}
	areFalseCount := 0
	for _, v := range r {
		if !v.is_something {
			areFalseCount++
		}
	}
	if areFalseCount != 1 {
		t.Error("Expected 1 false, got ", areFalseCount)
	}
}

func Test_IndexedList_SortByReadCount_EqualGroups(t *testing.T) {
	indexedList := listop.NewIndexedList[*testStruct]()
	test1 := &testStruct{ID: "1", is_something: true}
	test2 := &testStruct{ID: "2", is_something: false}
	test3 := &testStruct{ID: "3", is_something: false}
	test4 := &testStruct{ID: "4", is_something: true}
	test5 := &testStruct{ID: "5", is_something: true}

	indexedList.Set([]*testStruct{test1, test2, test3, test4, test5})
	r := indexedList.Gets([]string{"4", "3", "1", "2"})
	if len(r) != 4 {
		t.Error("Expected 4 elements, got ", len(r))
	}
	indexedList.Gets([]string{"1", "4", "3", "2"})
	indexedList.Gets([]string{"1", "4"})
	indexedList.Gets([]string{"1", "4"})
	indexedList.Gets([]string{"1", "4"})
	rcBf := indexedList.OrderedReadWriteCounts()
	indexedList.SortByReadCount()
	hasErr := false
	rc := indexedList.OrderedReadWriteCounts()
	ordered := indexedList.Pop(5)

	if ordered[0].ID != "5" {
		t.Error("Expected 5, got ", ordered[0].ID)
		hasErr = true
	}
	if !(ordered[1].ID == "3" || ordered[1].ID == "2") {
		t.Error("Expected 3 or 2, got ", ordered[0].ID)
		hasErr = true
	}
	if !(ordered[2].ID == "3" || ordered[2].ID == "2") {
		t.Error("Expected 3 or 2, got ", ordered[1].ID)
		hasErr = true
	}
	if !(ordered[3].ID == "1" || ordered[3].ID == "4") {
		t.Error("Expected 1 or 4, got ", ordered[2].ID)
		hasErr = true
	}
	if !(ordered[4].ID == "1" || ordered[4].ID == "4") {
		t.Error("Expected 1 or 4, got ", ordered[3].ID)
		hasErr = true
	}
	if hasErr {
		RCDisplay := ""
		for i := 0; i < len(rc); i++ {
			RCDisplay += strconv.Itoa(int(rc[i])) + " "
		}
		rcBfDisplay := ""
		for i := 0; i < len(rcBf); i++ {
			rcBfDisplay += strconv.Itoa(int(rcBf[i])) + " "
		}
		t.Log("Items read count: " + RCDisplay)
		t.Log("Items read count before sort: " + rcBfDisplay)
	}
}

type sortTestStruct uint32

func (s sortTestStruct) Index() string {
	return strconv.Itoa(int(s))
}

func Test_IndexedList_SortByReadCount_NormalyDistributed(t *testing.T) {
	indexedList := listop.NewIndexedList[sortTestStruct]()
	samples := []sortTestStruct{}
	for i := 0; i < 100; i++ {
		samples = append(samples, sortTestStruct(i))
	}
	indexedList.Set(samples)
	allIds := []string{}
	for i := 0; i < 100; i++ {
		allIds = append(allIds, strconv.Itoa(i))
	}
	for i := 0; i < 50; i++ {
		indexedList.Gets([]string{allIds[rand.Intn(len(allIds))], allIds[rand.Intn(len(allIds))]})
	}
	indexedList.SortByReadCount()
	ordered := indexedList.OrderedReadWriteCounts()
	isOrdered := true
	problemtaticIndex := 0
	for i := 0; i < 99; i++ {
		if int(ordered[i]) > int(ordered[i+1]) {
			isOrdered = false
			problemtaticIndex = i
			break
		}
	}
	if !isOrdered {
		t.Error("Not ordered at index ", problemtaticIndex)
		outputDisplay := ""
		for i := 0; i < len(ordered); i++ {
			outputDisplay += strconv.Itoa(int(ordered[i])) + " "
		}
		t.Log(outputDisplay)
	}

}
