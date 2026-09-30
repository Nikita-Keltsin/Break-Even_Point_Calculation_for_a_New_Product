package api

import (
	"log"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/handler"
	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartServer() {
	log.Println("Starting server")
	dsn := "host=localhost port=5433 user=postgres password=postgres dbname=breakeven_db sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	repo, err := repository.NewRepository(db)
	if err != nil {
		log.Println("ошибка инициализации репозитория:", err)
	}
	h := handler.NewHandler(repo)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/breakeven-point/feed", h.FeedHandler)
	r.GET("/breakeven-point/feed/:id", h.FeedHandler)
	r.GET("/breakeven-point/addition", h.AdditionHandler)
	r.GET("/breakeven-point/cost-types", h.CostTypesHandler)
	r.POST("/breakeven-point/draft", h.CreateDraftHandler)
	r.POST("/breakeven-point/publish", h.PublishHandler)
	r.POST("/breakeven-point/delete", h.DeleteHandler)
	r.POST("/breakeven-point/like", h.LikeHandler)

	r.Run()
	log.Println("Server down")
}