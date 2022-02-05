package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default

	r.GET("/test", func(val ...interface{}) gin.Handlefunc {
		return func(c *gin.Context) {
			// TODO
		}
	})

}
