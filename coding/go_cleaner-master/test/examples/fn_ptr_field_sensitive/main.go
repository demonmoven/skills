package main

type DoSportFunc func()

type SportPlayer interface {
	Choose(sport string)
	Play()
}

func main() {
	player := NewFootballPlayer()
	player.Choose("")
	player.Play()
}
