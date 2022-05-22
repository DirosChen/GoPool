package main

import (
	gopool "GoPool"
	"log"
	"time"
)

func main() {
	pool := gopool.MakePool(2)

	pool.Submit(func() {
		log.Println("one start")
		time.Sleep(2 * time.Second)
		log.Println("one end")
	})

	pool.Submit(func() {
		log.Println("tow start")
		time.Sleep(4 * time.Second)
		log.Println("tow end")
	})

	pool.Submit(func() {
		log.Println("three start")
	})

	pool.Submit(func() {
		log.Println("four start")
	})
	pool.SubmitDelay(func() {

	}, 3*time.Second)

	time.Sleep(3 * time.Hour)
}
