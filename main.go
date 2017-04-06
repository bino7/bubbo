package main

import (
	"fmt"
)

func main() {
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	fmt.Println(<-ch)
	fmt.Println(<-ch)
}

func f1(){
	i:=0
	m:=100.00
	for i<=90{
		m=m-m*0.0005
		i=i+1
		fmt.Println(i,m)
	}
	fmt.Println(i)
}


