package api

import (
	"log"
	"net/http"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/handler"
	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
)

func StartServer() {
	log.Println("Starting server")
	repo, err := repository.NewRepository()
	if err != nil {
		log.Println("ошибка инициализации репозитория:", err)
	}
	h := handler.NewHandler(repo)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/breakeven-point", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
	})
	r.GET("/breakeven-point/feed/:id", h.FeedHandler)
	r.POST("/breakeven-point/feed/:id/like", h.LikeHandler)
	r.GET("/breakeven-point/addition", h.AdditionHandler)
	r.POST("/breakeven-point/addition", h.AdditionSubmitHandler)
	r.GET("/breakeven-point/cost-types", h.CostTypesHandler)

	r.Run()
	log.Println("Server down")
}