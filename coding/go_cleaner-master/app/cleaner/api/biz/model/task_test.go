package model

import (
	"fmt"
	"testing"
)

func TestSubmit(t *testing.T) {
	(&Task{RepoName: "tiktok/item_core,tiktok/item_data"}).Submit()
	tasks := (&Task{}).TaskList(&TaskListCondition{})
	fmt.Println(tasks)
	tasks[0].Delete()
	tasks = (&Task{}).TaskList(&TaskListCondition{})
	fmt.Println(tasks)
}

func TestFetch(t *testing.T) {
	task := (&Task{TaskStatus: -1}).Fetch()
	fmt.Println(task)
}

func TestSubmitV2(t *testing.T) {
	(&Task{RepoName: "tiktok/item_data", Extra: `{"commit_branch":"chore/clean_worker_v2"}`}).Submit()
	tasks := (&Task{}).TaskList(&TaskListCondition{})
	fmt.Println(tasks)
}

func TestVersion(t *testing.T) {
	a := &Task{}
	v := a.FetchUnStartedWithVersion(nil)
	fmt.Println(v)
}
