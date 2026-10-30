package routes

import (
	"github.com/ConnieLWY/go-movie-streaming/Server/MagicStreamMoviesServer/controllers"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func SetupUnprotectedRoutes(router *gin.Engine, client *mongo.Client) {
	router.POST("/register", controllers.RegisterUser(client))
	router.POST("/login", controllers.LoginUser(client))
	router.POST("/logout", controllers.LogoutHandler(client))
	router.POST("/refresh", controllers.RefreshTokenHandler(client))
	router.GET("/movies", controllers.GetMovies(client))
	router.GET("/genres", controllers.GetGenres(client))
}
