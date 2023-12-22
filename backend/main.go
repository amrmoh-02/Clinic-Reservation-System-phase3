package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var client *mongo.Client

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type Doctor struct {
	ID       string   `json:"id" bson:"id"`
	DName    string   `json:"dname" bson:"dname"`
	Schedule []string `json:"schedule" bson:"schedule"`
}

type Patient struct {
	ID       string   `json:"id" bson:"id"`
	PName    string   `json:"pname" bson:"pname"`
	Schedule []string `json:"schedule" bson:"schedule"`
}

func main() {
	// Read environment variables
	dbBaseURL := os.Getenv("DB_BASE_URL")
	port := os.Getenv("PORT")

	fmt.Printf("DB Base URL: %s\n", dbBaseURL)
	fmt.Printf("Port: %s\n", port)

	if dbBaseURL == "" {
		log.Fatal("DB_BASE_URL environment variable not set")
	}
	if port == "" {
		port = "3000" // Default port if not provided
	}

	// Initialize MongoDB client
	ctx := context.TODO()
	clientOptions := options.Client().ApplyURI(dbBaseURL)
	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	// Verify MongoDB connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("MongoDB connection error: ", err)
	}

	fmt.Println("Connected to MongoDB!")

	// Initialize Gin router
	routes := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
	routes.Use(cors.New(config))

	// Set up routes
	routes.POST("/api/signup", SignUp)
	routes.GET("/api/doctors", GetDoctors)
	routes.GET("/api/doctors/:id", GetDoctorByID)
	routes.POST("/api/doctors", CreateDoctor)
	routes.PUT("/api/doctors/:id/schedule", SetDoctorSchedule)
	routes.POST("/api/patients/:id/appointments", BookAppointment)
	routes.PUT("/api/patients/:id/appointments/:appointmentID", UpdateAppointment)
	routes.DELETE("/api/patients/:id/appointments/:appointmentID", CancelAppointment)

	// Run the server
	routes.Run(":" + port)
}

func SignUp(c *gin.Context) {
	var newUser User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	if exists, err := isUsernameTaken(newUser.Username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking username availability"})
		return
	} else if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is already taken"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return
	}
	newUser.Password = string(hashedPassword)

	userCollection := client.Database("hospital").Collection("users")
	_, err = userCollection.InsertOne(context.Background(), newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}
func isUsernameTaken(username string) (bool, error) {
	userCollection := client.Database("hospital").Collection("users")
	count, err := userCollection.CountDocuments(context.Background(), bson.M{"username": username})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetDoctors(c *gin.Context) {
	coll := client.Database("hospital").Collection("doctor")
	cur, err := coll.Find(context.Background(), bson.D{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching doctor data"})
		return
	}
	defer cur.Close(context.Background())

	var doctors []Doctor
	for cur.Next(context.Background()) {
		var doctor Doctor
		if err := cur.Decode(&doctor); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding doctor data"})
			return
		}
		doctors = append(doctors, doctor)
	}

	c.JSON(http.StatusOK, doctors)
}

func GetDoctorByID(c *gin.Context) {
	doctorID := c.Param("id")

	coll := client.Database("hospital").Collection("doctor")
	filter := bson.M{"id": doctorID}

	var doctor Doctor
	err := coll.FindOne(context.Background(), filter).Decode(&doctor)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Doctor not found"})
		return
	}

	c.JSON(http.StatusOK, doctor)
}

func CreateDoctor(c *gin.Context) {
	var newDoctor Doctor
	if err := c.ShouldBindJSON(&newDoctor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	coll := client.Database("hospital").Collection("doctor")
	_, err := coll.InsertOne(context.Background(), newDoctor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating doctor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Doctor created successfully"})
}

func SetDoctorSchedule(c *gin.Context) {
	doctorID := c.Param("id")

	var schedule []string
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	coll := client.Database("hospital").Collection("doctor")
	filter := bson.M{"id": doctorID}
	update := bson.M{"$set": bson.M{"schedule": schedule}}

	_, err := coll.UpdateOne(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating doctor's schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Doctor's schedule updated successfully"})
}

func GetPatientAppointments(c *gin.Context) {
	patientID := c.Param("id")

	coll := client.Database("hospital").Collection("patients")
	filter := bson.M{"id": patientID}

	var patient Patient
	err := coll.FindOne(context.Background(), filter).Decode(&patient)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Patient not found"})
		return
	}

	c.JSON(http.StatusOK, patient.Schedule)
}

func BookAppointment(c *gin.Context) {
	patientID := c.Param("id")

	var newAppointment string
	if err := c.ShouldBindJSON(&newAppointment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	coll := client.Database("hospital").Collection("patients")
	filter := bson.M{"id": patientID}
	update := bson.M{"$push": bson.M{"schedule": newAppointment}}

	_, err := coll.UpdateOne(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error booking appointment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment booked successfully"})
}

func UpdateAppointment(c *gin.Context) {
	patientID := c.Param("id")
	appointmentID := c.Param("appointmentID")

	var updatedAppointment string
	if err := c.ShouldBindJSON(&updatedAppointment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	coll := client.Database("hospital").Collection("patients")
	filter := bson.M{"id": patientID, "schedule": appointmentID}
	update := bson.M{"$set": bson.M{"schedule.$": updatedAppointment}}

	_, err := coll.UpdateOne(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating appointment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment updated successfully"})
}

func CancelAppointment(c *gin.Context) {
	patientID := c.Param("id")
	appointmentID := c.Param("appointmentID")

	coll := client.Database("hospital").Collection("patients")
	filter := bson.M{"id": patientID}
	update := bson.M{"$pull": bson.M{"schedule": appointmentID}}

	_, err := coll.UpdateOne(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error canceling appointment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment canceled successfully"})
}
