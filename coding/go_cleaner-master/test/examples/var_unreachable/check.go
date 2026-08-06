// This case will panic if the unreachable statement is not removed.
// Add 'output_changes' to make the error message shorter.
// RUN: %cleaner -f --output_changes | %filecheck %s
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
	} else {
		t.Do = playBasketball
		return
	}

	t.NeverDo = takeRest
}

func main() {
	player := NewFootballPlayer()
	player.Choose("football")
	player.Play()
}

// CHECK: func (t *FootBallPlayer) Choose(sport string) {
// CHECK-NEXT: 	if sport == "football" {
// CHECK-NEXT: 		t.Do = playFootball
// CHECK-NEXT: 		return
// CHECK-NEXT: 	} else {
// CHECK-NEXT: 		t.Do = playBasketball
// CHECK-NEXT: 		return
// CHECK-NEXT: 	}
// CHECK-NEXT: }
