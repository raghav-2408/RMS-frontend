package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Customer struct {
	Name         string   `bson:"name"`
	Phone        string   `bson:"phone"`
	OrderedItems []string `bson:"orderedItems"`
	TotalAmount  float64  `bson:"totalAmount"`
}

type MenuItem struct {
	Name  string
	Price float64
}

var menuItems = []MenuItem{
	{"Burger", 50.00},
	{"Pizza", 150.00},
	{"Pasta", 100.00},
	{"Sandwich", 40.00},
	{"Fries", 30.00},
	{"Soda", 20.00},
	{"Coffee", 25.00},
	{"Salad", 60.00},
	{"Ice Cream", 45.00},
	{"Soup", 35.00},
}

var client *mongo.Client

func ConnectDB() *mongo.Client {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	fmt.Println("Connected to MongoDB!")
	return client
}

func GetCustomers() []Customer {
	customersCollection := client.Database("restaurant").Collection("customers")
	cursor, err := customersCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal("Error retrieving customers:", err)
	}
	defer cursor.Close(context.TODO())

	var customers []Customer

	for cursor.Next(context.TODO()) {
		var customer Customer
		if err = cursor.Decode(&customer); err != nil {
			log.Fatal(err)
		}
		customers = append(customers, customer)
	}
	return customers
}

func CalculateTotal(orderedItems []string) float64 {
	total := 0.0
	for _, itemName := range orderedItems {
		for _, menuItem := range menuItems {
			if strings.TrimSpace(itemName) == menuItem.Name {
				total += menuItem.Price
			}
		}
	}
	return total
}

func RenderTemplate(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous">
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js" integrity="sha384-YvpcrYf0tY3lHB60NNkmXc5s9fDVZLESaAA55NDzOxhy9GkcIdslK1eN7N6jIeHz" crossorigin="anonymous"></script>
    <title>Restaurant Ordering System</title>
    <style>
        body { font-family: Arial, sans-serif; background-color: #E5E5E5; margin: 0; padding: 0; }
        .container { max-width: 800px; margin: 40px auto; padding: 30px; background: white; box-shadow: 0 2px 10px rgba(0,0,0,0.1); border-radius: 8px; }
        h1, h2 { color: #333; text-align: center; margin: 20px 0; }
        .customer-details { display: none; }
        .menu-card { position: fixed; bottom: 40px; right: 30px; width: 200px; padding: 10px; background: #fff; box-shadow: 0 2px 5px rgba(0,0,0,0.3); border-radius: 8px; text-align: center; }
        .menu-card h2 { font-size: 1em; margin: 0; color: #007bff; }
        .menu-card ul { padding: 0; margin: 10px 0; list-style: none; }
    </style>
    <script>
        function toggleCustomerDetails() {
            var details = document.getElementById("customerDetails");
            details.style.display = (details.style.display === "none" || details.style.display === "") ? "block" : "none";
        }
    </script>
</head>
<body>
    <div class="container">
        <h1>Welcome to the Restaurant Management System!</h1>
        
		<center>
        <button class="btn btn-primary" onclick="toggleCustomerDetails()">View Customers</button>
        </center>
        <div id="customerDetails" class="customer-details mt-4">
            <h2>Customer Orders</h2>
            <ul>
                {{range .Customers}}
                <li class="mb-3">
                    <strong>Name :</strong> {{.Name}} <br>
                    <strong>Total :</strong> Rs {{printf "%.2f" .TotalAmount}} <br>
                  <strong>Ordered Items:</strong> 
					{{- if eq (len .OrderedItems) 1 -}}
						{{index .OrderedItems 0}}
					{{- else -}}
						{{range $index, $item := .OrderedItems}}
							{{if $index}}, {{end}}{{$item}}
						{{end}}
					{{- end}}
                </li>
                <hr>
                {{end}}
            </ul>
        </div>

        <form action="/add-customer" method="POST" class="mt-4">
            <label>Name</label>
            <input type="text" class="form-control" name="name" required>
            <label>Phone</label>
            <input type="text" class="form-control" name="phone" required>
            <label>Order Items (comma-separated)</label>
            <input type="text" class="form-control" name="orderedItems" required>
            <center><button type="submit" class="btn btn-success mt-3">Place Order</button></center>
        </form>

        <div class="footer mt-4 text-center">&copy; {{.Year}} Restaurant Management System</div>
    </div>

    <div class="menu-card">
<h2 style="background: grey; color: white;">Menu</h2>
        <ul>
            {{range .MenuItems}}
                <li>{{.Name}} - Rs {{printf "%.2f" .Price}}</li>
                <hr>
            {{end}}
        </ul>
    </div>
</body>
</html>
`

	data := struct {
		MenuItems []MenuItem
		Customers []Customer
		Year      int
	}{
		MenuItems: menuItems,
		Customers: GetCustomers(),
		Year:      2024,
	}

	tmplParsed := template.Must(template.New("webpage").Parse(tmpl))
	err := tmplParsed.Execute(w, data)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

func AddCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		phone := r.FormValue("phone")
		orderedItems := strings.Split(r.FormValue("orderedItems"), ",")
		totalAmount := CalculateTotal(orderedItems)

		customer := Customer{
			Name:         name,
			Phone:        phone,
			OrderedItems: orderedItems,
			TotalAmount:  totalAmount,
		}

		_, err = client.Database("restaurant").Collection("customers").InsertOne(context.TODO(), customer)
		if err != nil {
			http.Error(w, "Error saving customer", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func main() {
	client = ConnectDB()
	defer client.Disconnect(context.TODO())

	http.HandleFunc("/", RenderTemplate)
	http.HandleFunc("/add-customer", AddCustomer)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
