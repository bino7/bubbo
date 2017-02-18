package main

import (
	"fmt"
)
func boo(s string)(n string){
	fmt.Println(s,n)
	return
}
func foo(s string)(n string){
	n=s
	boo(s)
	return
}

type Detail map[string]interface{}

func main() {
	d:=Detail{"Type":"AAA","BBB":"BBB"}
	fmt.Println(d["Type"])
}


