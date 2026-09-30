package api

import (
	"log"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartServer() {
	log.Println("Starting server (lab3 API)")

	dsn := "host=localhost port=5433 user=postgres password=postgres dbname=breakeven_db sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	repo, err := repository.NewRepository(db)
	if err != nil {
		log.Println("ошибка инициализации репозитория:", err)
	}
	mc, err := NewMinIOClient()
	if err != nil {
		log.Fatal("Ошибка MinIO:", err)
	}
	if err := EnsureBucket(mc); err != nil {
		log.Println("предупреждение по бакету:", err)
	}
	a := &API{Repo: repo, MC: mc}

	r := gin.Default()
	apiR := r.Group("/api")
	{
		apiR.GET("/costs", a.ListCosts)
		apiR.GET("/costs/feed", a.FeedCost)
		apiR.GET("/costs/feed/:id", a.FeedCost)
		apiR.GET("/costs/draft", a.GetDraft)
		apiR.POST("/costs", a.CreateCost)
		apiR.PUT("/costs/:id/publish", a.PublishCost)
		apiR.DELETE("/costs/:id", a.DeleteCost)
		apiR.POST("/costs/:id/like", a.LikeCost)
		apiR.POST("/users/register", a.Register)
		apiR.POST("/users/login", a.Login)
		apiR.POST("/users/logout", a.Logout)
	}

	r.Run()
	log.Println("Server down")
}