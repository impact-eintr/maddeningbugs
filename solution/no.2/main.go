package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default

	s := "test"

	r.GET("/test", func(val string) gin.Handlefunc {
		return func(c *gin.Context) {
			fmt.Println(val)
		}
	}(s))

}
