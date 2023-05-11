package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson" //this will help us to create id  , every id will be a bson id , mongo db ko bson hi aati hai smjh
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName //in this project we are using localhost mongod //means its installed in our pc / it'll be done inna port & mongo uses 27017 as its port

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name" `
	Salary float64 `json: "salary"`
	Age    float64 `json: "age"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))      //ye krdiya client ko define (alias) , ab baar baar itna bada mongo.nEW..  nahi likhna padega bss client likhna hoga
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) //by default mongo has blocking functions which may block something so we can add a Timeout to it
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	//how are we talking to mongo db in our prgm? -> by using mongoInstance .
	//mongoiNSTANCE ki andar hi hai db -> uske andar hai collection-> uske andar hai tables.
	mg = MongoInstance{ //MongoInstance struct banaya hai upar aur uski Client & Db ki value set krdi aur client aur db ko upar define krdiya
		Client: client,
		Db:     db,
	}
	return nil
}

func main() {

	if err := Connect(); err != nil {
		log.Fatal(err)
	}

	app := fiber.New() //jaise nodejs ke saath we have to use express js , similarly golang ke saath fiber //fiber is more than 10x faster than express

	app.Get("/employee", func(c *fiber.Ctx) error { //ye wala Get - to get list of all the employees.

		query := bson.D{{}} //in do brackets ke andar we define the query , since we want list of all the employees we'll leave that empty

		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query) //Find fn is given by Mongo , in which we pass the context & the query
		//toh mg(mongoInstance) ke andar jaake -> db mein jaake )-> collections mein jaake -> tables mein chalega Find -> aur {{}} matlb all tables
		//this we are storing in our cursor
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var employees []Employee = make([]Employee, 0) //make command is used to initialize only the slices , maps & channels
		//means Employee datatype ka ek slice bana liya initialised with 0

		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		} // this means that wtv data we receive in our cursor , this is going to convert it into structs that are understandable// thats why we pass the employees

		return c.JSON(employees)

	}) //jab iss route pe aayein(/employee) you want a function to be called which returns an err , func has a defintion -> it takes in c which gives you fiber.context

	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")
		employee := new(Employee) //suppose user has requested a new employee's data & he sends his name age salary .. to this api & this api now needs to READ that information

		//suppose user has requested a new employee's data & he sends his name age salary .. to this api & this api now needs to READ that information
		//which it does through context.BodyParser
		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		} //aur iske andar employee pass kiya means all the data become formatted in the format that you need

		//now we want mongodb to create its own id
		employee.ID = ""                                                    // this forces mongodb to create its own id
		insertionResult, err := collection.InsertOne(c.Context(), employee) //InsertOne is a mongoDb function -> table mein ek entrt insert krra hai

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}} //and using this id we'll create a query & find the record which has this id. to check what we insert rn using insertedONe was  in the db or not

		createdRecord := collection.FindOne(c.Context(), filter) // collection jo already db mein tha wo mil gya

		createdEmployee := &Employee{}        // ab usko Employee datatype ke structure mein convert krne ke liye
		createdRecord.Decode(createdEmployee) //ye krna pada

		return c.Status(201).JSON(createdEmployee) //201 sends a msg -> new resource created ,.

	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id") //you can have acces to id very easily through c.params //jo ki string mein hogi .

		//Package primitive contains types similar to Go primitives for BSON types
		//ObjectIDFromHex creates a new ObjectID from a hex string. It returns an error if the hex string is not a valid ObjectID.
		employeeID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		update := bson.D{
			//$set use krke data set krte hai sbse pehle in mongodb  -> key value hoti hai inside bson
			//key is $set  & value is whats inside your employee
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		//in mongo ->db.collection.findOneAndUpdate() updates the first matching document in the collection that matches the filter
		err = mg.Db.Collection("employee").FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}

		employee.ID = idParam //firse string bana diya?
		//status code 200 is sucess OK
		return c.Status(200).JSON(employee)

	})
	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))

		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)

		if err != nil {
			return c.SendStatus(500) //unexpected condition
		}

		if result.DeletedCount < 1 { //mtlb delete hua hi nahi
			return c.SendStatus(404) //means server could not found client requested webpage // ya jo client delte krna chaha , server could't . so 404
		}

		return c.Status(200).JSON("record deleted")

	})

	log.Fatal(app.Listen(":3000"))
}
