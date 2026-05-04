package main

import (
	"fmt"
	"log"
	"os"
)



func main(){
	filePath := "./test.txt"
	fileBuffer,err := os.ReadFile(filePath)
	
	if err!=nil{
		log.Fatal(err)	
	}
		
	fmt.Println(string(fileBuffer))
}

