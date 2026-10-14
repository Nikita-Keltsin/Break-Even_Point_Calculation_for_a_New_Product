package api

import (
	"context"
	"log"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/handler"
	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	_ "Break-Even_Point_Calculation_for_a_New_Product/docs" // сгенерировано swag init

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	// ЛР-4: Redis — чёрный список отозванных JWT
	rdb := NewRedisClient()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Ошибка Redis:", err)
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

	// ---------- ЛР-3 + ЛР-4: веб-сервис REST API (10 методов) с авторизацией ----------
	a := &API{Repo: repo, MC: mc, RDB: rdb}
	apiR := r.Group("/api")
	{
		bep := apiR.Group("/breakeven_point") // домен breakeven_point (услуга — вид затрат)
		// гость и создатель: только чтение (с токеном заполняются like и is_mine)
		bep.GET("", a.WithOptionalAuth, a.ListCosts)
		bep.GET("/feed", a.WithOptionalAuth, a.FeedCost)
		bep.GET("/feed/:id", a.WithOptionalAuth, a.FeedCost)
		// только создатель (JWT обязателен, иначе 401)
		creator := bep.Group("", a.WithAuth)
		creator.GET("/draft", a.GetDraft)
		creator.POST("", a.CreateCost)
		creator.PUT("/:id/publish", a.PublishCost)
		creator.DELETE("/:id", a.DeleteCost)
		creator.POST("/:id/like", a.LikeCost)

		users := apiR.Group("/users") // домен users
		users.POST("/register", a.Register)
		users.POST("/login", a.Login)
		users.POST("/logout", a.WithAuth, a.Logout)
	}

	// ЛР-4: Swagger — http://localhost:8080/swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run() // :8080
	log.Println("Server down")
}
