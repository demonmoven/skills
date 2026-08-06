// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

import "fmt"

type DoSportFakelyFunc func()

type FakeFootBallPlayer struct {
	Do      DoSportFakelyFunc
	NeverDo DoSportFunc
}

func NewFakeFootballPlayer() SportPlayer {
	return &FakeFootBallPlayer{}
}

func (t *FakeFootBallPlayer) Play() {
	t.Do()
}

func takeRest() {
	fmt.Println("I'm taking a rest")
}

func playFootballReally() {
	fmt.Println("I'm really playing football")
}

func (t *FakeFootBallPlayer) Choose(sport string) {
	if sport == "football" {
		t.Do = takeRest
		t.NeverDo = playFootballReally
	}
}

// CHECK: func (t *FakeFootBallPlayer) Play() {
// CHECK-NEXT: 	t.Do()
// CHECK-NEXT: }
// CHECK-NEXT:
// CHECK-NEXT: func takeRest() {
// CHECK-NEXT: 	fmt.Println("I'm taking a rest")
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func (t *FakeFootBallPlayer) Choose(sport string) {
// CHECK-NEXT: 	if sport == "football" {
// CHECK-NEXT: 		t.Do = takeRest
// CHECK-NEXT: 	}
// CHECK-NEXT: }
