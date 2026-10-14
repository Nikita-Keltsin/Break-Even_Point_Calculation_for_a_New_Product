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
	mc, err := NewMinIOClient()
	if err != nil {
		log.Fatal("Ошибка MinIO:", err)
	}
	if err := EnsureBucket(mc); err != nil {
		log.Println("предупреждение по бакету:", err)
	}

	r := gin.Default()

	// ---------- ЛР-2: SSR-страницы (6 HTTP-методов) ----------
	h := handler.NewHandler(repo)
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/breakeven-point/cost-types", h.CostTypesHandler)
	r.GET("/breakeven-point/feed", h.FeedHandler)
	r.GET("/breakeven-point/feed/:id", h.FeedHandler)
	r.GET("/breakeven-point/addition", h.AdditionHandler)
	r.POST("/breakeven-point/draft", h.CreateDraftHandler)
	r.POST("/breakeven-point/publish", h.PublishHandler)
	r.POST("/breakeven-point/delete", h.DeleteHandler)

	// ---------- ЛР-3: веб-сервис REST API (10 методов) ----------
	a := &API{Repo: repo, MC: mc}
	apiR := r.Group("/api")
	{
		bep := apiR.Group("/breakeven_point") // домен breakeven_point (услуга — вид затрат)
		bep.GET("", a.ListCosts)
		bep.GET("/feed", a.FeedCost)
		bep.GET("/feed/:id", a.FeedCost)
		bep.GET("/draft", a.GetDraft)
		bep.POST("", a.CreateCost)
		bep.PUT("/:id/publish", a.PublishCost)
		bep.DELETE("/:id", a.DeleteCost)
		bep.POST("/:id/like", a.LikeCost)

		users := apiR.Group("/users") // домен users
		users.POST("/register", a.Register)
		users.POST("/login", a.Login)
		users.POST("/logout", a.Logout)
	}

	r.Run() // :8080
	log.Println("Server down")
}
