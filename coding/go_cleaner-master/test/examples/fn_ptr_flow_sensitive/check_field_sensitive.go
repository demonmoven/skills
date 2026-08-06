// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
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
	t.Do = playBasketball
}

// CHECK: func playBasketball() {
// CHECK-NEXT:     fmt.Println("I'm playing basketball")
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func (t *FootBallPlayer) Choose(sport string) {
// CHECK-NEXT:     t.Do = playBasketball
// CHECK-NEXT: }
