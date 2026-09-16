package main

import (
	"log"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/api"
)

func main() {
	log.Println("Application start!")
	api.StartServer()
	log.Println("Application terminated!")
}