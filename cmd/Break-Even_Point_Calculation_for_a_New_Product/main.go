package main

import (
	"log"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/api"
)

// @title           breakeven_point API
// @version         1.0
// @description     Веб-сервис расчёта точки безубыточности нового продукта: виды затрат (домен breakeven_point) и пользователи (домен users).
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Введите: Bearer <access_token из POST /api/users/login>
func main() {
	log.Println("Application start!")
	api.StartServer()
	log.Println("Application terminated!")
}
