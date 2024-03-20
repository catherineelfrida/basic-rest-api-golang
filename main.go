package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"
)

type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Order struct {
	OrderID      uint      `json:"orderId" gorm:"primaryKey"`
	CustomerName string    `json:"customerName"`
	OrderedAt    time.Time `json:"orderedAt"`
	Items        []Item    `json:"items" gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Item struct {
	ItemID      uint   `json:"lineItemId" gorm:"primaryKey"`
	ItemCode    string `json:"itemCode"`
	Description string `json:"description"`
	Quantity    uint   `json:"quantity"`
	OrderID     uint   `json:"-"`
}

var users []User

func main() {
	db, err := connectDB()
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&User{}, &Order{}, &Item{})

	GinHttp()
}

func GinHttp() {
	// gin => framework HTTP punya golang
	// big community
	engine := gin.New()

	// serve static template
	// engine.LoadHTMLGlob("static/*")
	engine.Static("/static", "./static")

	engine.LoadHTMLGlob("template/*")
	engine.GET("/template/index/:name", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.tmpl", map[string]interface{}{
			"title": ctx.Param("name"),
		})
	})

	// membuat prefix group
	v1 := engine.Group("/api/v1")
	{
		usersGroup := v1.Group("/users")
		{
			// [GET] /api/v1/users
			// filter user by email
			usersGroup.GET("", func(ctx *gin.Context) {
				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				users := []User{}
				if err := db.Find(&users).Error; err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to get users",
					})
					return
				}

				email := ctx.Query("email")
				if email != "" {
					filterUsers := []User{}

					if err := db.Where("email LIKE ?", "%"+email+"%").Find(&filterUsers).Error; err != nil {
						ctx.JSON(http.StatusInternalServerError, gin.H{
							"message": "failed to get users",
						})
						return
					}

					ctx.JSON(http.StatusOK, filterUsers)
					return
				}
				ctx.JSON(http.StatusOK, users)
			})

			// [POST] /api/v1/users
			usersGroup.POST("", func(ctx *gin.Context) {
				// binding payload
				user := User{}
				if err := ctx.BindJSON(&user); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "failed to bind body",
					})
					return
				}

				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				if err := db.Create(&user).Error; err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to create user",
					})
					return
				}

				ctx.JSON(http.StatusCreated, gin.H{
					"message": "user created",
				})
			})

			// [GET] /api/v1/users/:id
			usersGroup.GET("/:id", func(ctx *gin.Context) {
				id, err := strconv.Atoi(ctx.Param("id"))
				if err != nil || id <= 0 {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "invalid ID",
					})
					return
				}

				user := User{}
				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				if err := db.Where("id = ?", id).First(&user).Error; err != nil {
					ctx.JSON(http.StatusNotFound, gin.H{
						"message": "user not found",
					})
					return
				}

				ctx.JSON(http.StatusOK, user)
			})

			// [PUT] /api/v1/users/:id
			usersGroup.PUT("/:id", func(ctx *gin.Context) {
				id, err := strconv.Atoi(ctx.Param("id"))
				if err != nil || id <= 0 {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "invalid ID",
					})
					return
				}

				user := User{}
				if err := ctx.BindJSON(&user); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "failed to bind body",
					})
					return
				}

				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				if err := db.Where("id = ?", id).Updates(&user).Error; err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to update user",
					})
					return
				}

				ctx.JSON(http.StatusOK, gin.H{
					"message": "user updated",
				})
			})

			// [DELETE] /api/v1/users/:id
			usersGroup.DELETE("/:id", func(ctx *gin.Context) {
				id, err := strconv.Atoi(ctx.Param("id"))
				if err != nil || id <= 0 {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "invalid ID",
					})
					return
				}

				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				if err := db.Where("id = ?", id).Delete(&User{}).Error; err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to delete user",
					})
					return
				}

				ctx.JSON(http.StatusOK, gin.H{
					"message": "user deleted",
				})
			})
		}

		orderGroup := v1.Group("/orders")
		{
			orderGroup.GET("", func(ctx *gin.Context) {
				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				orders := []Order{}
				if err := db.Preload("Items").Find(&orders).Error; err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to get orders",
					})
					return
				}

				ctx.JSON(http.StatusOK, orders)
			})

			orderGroup.POST("", func(ctx *gin.Context) {
				order := Order{}
				if err := ctx.BindJSON(&order); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "failed to bind body",
					})
					return
				}

				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				if err := db.Create(&order).Error; err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to create order",
					})
					return
				}

				ctx.JSON(http.StatusCreated, gin.H{
					"message": "order created",
				})
			})

			orderGroup.GET("/:id", func(ctx *gin.Context) {
				id, err := strconv.Atoi(ctx.Param("id"))
				if err != nil || id <= 0 {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "invalid order ID",
					})
					return
				}

				order := Order{}
				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				if err := db.Preload("Items").Where("order_id = ?", id).First(&order).Error; err != nil {
					ctx.JSON(http.StatusNotFound, gin.H{
						"message": "order not found",
					})
					return
				}

				ctx.JSON(http.StatusOK, order)
			})

			orderGroup.PUT("/:id", func(ctx *gin.Context) {
				id, err := strconv.Atoi(ctx.Param("id"))
				if err != nil || id <= 0 {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "invalid ID",
					})
					return
				}

				order := Order{}
				if err := ctx.BindJSON(&order); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "failed to bind order data",
					})
					return
				}

				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				if err := db.Model(&Order{}).Where("order_id = ?", id).Updates(&order).Error; err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to update order",
					})
					return
				}

				for _, item := range order.Items {
					if err := db.Model(&Item{}).Where("item_id = ?", item.ItemID).Updates(&item).Error; err != nil {
						ctx.JSON(http.StatusInternalServerError, gin.H{
							"message": "failed to update item",
						})
						return
					}
				}

				ctx.JSON(http.StatusOK, gin.H{
					"message": "order and items updated",
				})
			})

			orderGroup.DELETE("/:id", func(ctx *gin.Context) {
				id, err := strconv.Atoi(ctx.Param("id"))
				if err != nil || id <= 0 {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "invalid ID",
					})
					return
				}

				db, err := connectDB()
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to connect to database",
					})
					return
				}

				tx := db.Begin()

				if err := tx.Where("order_id = ?", id).Delete(&Item{}).Error; err != nil {
					tx.Rollback()
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to delete items related to order",
					})
					return
				}

				if err := tx.Where("order_id = ?", id).Delete(&Order{}).Error; err != nil {
					tx.Rollback()
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"message": "failed to delete order",
					})
					return
				}

				tx.Commit()

				ctx.JSON(http.StatusOK, gin.H{
					"message": "order and related items deleted",
				})
			})
		}

		engine.Run(":80")
	}
}

func connectDB() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=admin dbname=orders_by port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func NetHttp() {
	// /users => API path
	// func(w http.ResponseWriter, r *http.Request) => handler function
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		// validasi request payload (header, body)
		// memanggil logic
		// memberkan response

		// get all users
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(users)
			return
		}
		// create user
		if r.Method == http.MethodPost {
			user := User{}
			// only bind username and email
			if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			user.ID = uint(len(users) + 1)
			users = append(users, user)

			w.WriteHeader(http.StatusAccepted)
			return
		}

		// mini quiz
		// buatlah method
		// PUT /users/:id untuk edit user by id
		// Delete /users/:id untuk delete user by id
	})

	// {id} => path variable
	http.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			pathValue, _ := strconv.Atoi(r.PathValue("id"))
			for _, user := range users {
				if user.ID == uint(pathValue) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(user)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// :8080 PORT
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}
