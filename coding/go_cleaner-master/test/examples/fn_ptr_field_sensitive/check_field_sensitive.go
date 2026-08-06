// RUN: %cleaner -f --output_changes | %filecheck %s
package main

import "fmt"

type FootBallPlayer struct {
	Do      DoSportFunc
	NeverDo DoSportFunc
}

func NewFootballPlayer() SportPlayer {
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

func (t *FootBallPlayer) Choose(sport string) {
	t.Do = playFootball
	t.NeverDo = playBasketball
}

// CHECK: func playFootball() {
// CHECK-NEXT:     fmt.Println("I'm playing football")
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func (t *FootBallPlayer) Choose(sport string) {
// CHECK-NEXT:     t.Do = playFootball
// CHECK-NEXT: }
