package main

import clc "github.com/morganhein/centurylinkchallenge"

func main() {
	e := clc.StartTheChallenge()
	if e != nil {
		panic(e)
	}
}
