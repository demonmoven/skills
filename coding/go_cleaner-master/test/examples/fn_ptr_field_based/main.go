package main

type DoSportFunc func()

type SportPlayer interface {
	Choose(sport string)
	Play()
}

func main() {
	player := NewFakeFootballPlayer()
	player.Choose("")
	player.Play()
}
