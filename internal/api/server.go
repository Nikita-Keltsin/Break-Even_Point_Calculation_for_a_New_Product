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
	log.Println("Starting server breakeven_point")

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

	// ЛР-2: ровно 6 HTTP-методов — 3 GET, 2 POST через ORM, 1 POST удаления через SQL UPDATE
	r.GET("/breakeven-point/cost-types", h.CostTypesHandler) // плитка + поиск (ORM)
	r.GET("/breakeven-point/feed", h.FeedHandler)            // лента: одна строка из БД (ORM)
	r.GET("/breakeven-point/feed/:id", h.FeedHandler)        //   тот же GET-метод ленты, по id
	r.GET("/breakeven-point/addition", h.AdditionHandler)    // добавление: шаг 1 / шаг 2 (ORM)
	r.POST("/breakeven-point/draft", h.CreateDraftHandler)   // «Далее» — черновик (ORM)
	r.POST("/breakeven-point/publish", h.PublishHandler)     // «Опубликовать» — смена статуса (ORM)
	r.POST("/breakeven-point/delete", h.DeleteHandler)       // логическое удаление (SQL UPDATE, без ORM)

	r.Run() // :8080
	log.Println("Server down")
}
