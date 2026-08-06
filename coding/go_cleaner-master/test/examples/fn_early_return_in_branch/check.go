// RUN: %cleaner -f --passes=clean-after-return
// RUN: cat %s | %filecheck %s
package main

import "fmt"

type DoSportFunc func()

type FootBallPlayer struct {
	Do      DoSportFunc
	NeverDo DoSportFunc
}

func NewFootballPlayer() *FootBallPlayer {
	return &FootBallPlayer{}
}

func (t *FootBallPlayer) Play() {
	t.Do()
}

func playFootball() {
	fmt.Println("I'm playing football")
}

func playBasketball() {
	fmt.Println("I'm playing basketball")
}

func takeRest() {
	fmt.Println("I'm taking a rest")
}

func (t *FootBallPlayer) Choose(sport string) {
	if sport == "football" {
		t.Do = playFootball
		return
		t.NeverDo = takeRest
	} else {
		t.Do = playBasketball
		return
	}
}

func main() {
	player := NewFootballPlayer()
	player.Choose("football")
	player.Play()
}

// CHECK: func playBasketball() {
// CHECK-NEXT: 	fmt.Println("I'm playing basketball")
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func (t *FootBallPlayer) Choose(sport string) {
// CHECK-NEXT: 	if sport == "football" {
// CHECK-NEXT: 		t.Do = playFootball
// CHECK-NEXT: 		return
// CHECK-EMPTY:
// CHECK-NEXT: 	} else {
// CHECK-NEXT: 		t.Do = playBasketball
// CHECK-NEXT: 		return
// CHECK-NEXT: 	}
// CHECK-NEXT: }
