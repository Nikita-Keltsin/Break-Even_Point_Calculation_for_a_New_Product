package api

import (
	"log"

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

	r.GET("/breakeven-point/cost-types", h.CostTypesHandler)
	r.GET("/breakeven-point/feed/:id", h.FeedHandler)
	r.GET("/breakeven-point/addition", h.AdditionHandler)
	r.GET("/breakeven-point/request/:id", h.RequestHandler)
	r.POST("/breakeven-point/addition", h.AdditionSubmitHandler)
	r.POST("/breakeven-point/feed/:id/like", h.LikeHandler)
	r.POST("/breakeven-point/request/add", h.AddToRequestHandler)
	r.POST("/breakeven-point/request/update", h.UpdateLinkHandler)
	r.POST("/breakeven-point/request/:id/delete", h.DeleteRequestHandler)

	r.Run()
	log.Println("Server down")
}