package main

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
)

type UberPrices struct {
	Prices []struct {
		ProductID       string  `json:"product_id"`
		CurrencyCode    string  `json:"currency_code"`
		DisplayName     string  `json:"display_name"`
		Estimate        string  `json:"estimate"`
		LowEstimate     int     `json:"low_estimate"`
		HighEstimate    int     `json:"high_estimate"`
		SurgeMultiplier int     `json:"surge_multiplier"`
		Duration        int     `json:"duration"`
		Distance        float64 `json:"distance"`
	} `json:"prices"`
}

type UberEstimates struct {
	Times []struct {
		LocalizedDisplayName string `json:"localized_display_name"`
		Estimate             int    `json:"estimate"`
		DisplayName          string `json:"display_name"`
		ProductID            string `json:"product_id"`
	} `json:"times"`
}

type Request struct {
	StartPt      string   `json:"starting_from_location_id"`
	Location_ids []string `json:"location_ids"`
}

type Response struct {
	TripId           string   `json:"id" bson:"_id"`
	Status           string   `json:"status" bson:"status"`
	StartPt          string   `json:"starting_from_location_id" bson:"starting_from_location_id"`
	Bestlocation_ids []string `json:"best_route_location_ids" bson:"best_route_location_ids"`
	Costs            int      `json:"total_uber_costs" bson:"total_uber_costs"`
	Duration         int      `json:"total_uber_duration" bson:"total_uber_duration"`
	Distance         float64  `json:"total_distance" bson:"total_distance"`
}

type PutResponse struct {
	TripId           string   `json:"id" bson:"_id"`
	Status           string   `json:"status" bson:"status"`
	StartPt          string   `json:"starting_from_location_id" bson:"starting_from_location_id"`
	NextPt           string   `json:"next_destination_location_id" bson:"next_destination_location_id"`
	Bestlocation_ids []string `json:"Bestlocation_ids" bson:"Bestlocation_ids"`
	Costs            int      `json:"total_uber_costs" bson:"total_uber_costs"`
	Duration         int      `json:"total_uber_duration" bson:"total_uber_duration"`
	Distance         float64  `json: "total_distance" bson:"total_distance"`
	ETA              int      `json: "uber_wait_time_eta" bson:"uber_wait_time_eta"`
}

type MongoResponse struct {
	Id         bson.ObjectId `json:"id" bson:"_id"`
	Name       string        `json:"name" bson:"name"`
	Address    string        `json:"address" bson:"address" `
	City       string        `json:"city"  bson:"city"`
	State      string        `json:"state"  bson:"state"`
	ZipCode    string        `json:"zip"  bson:"zip" `
	Coordinate struct {
		Lat float64 `json:"lat"   bson:"lat"`
		Lng float64 `json:"lng"   bson:"lng"`
	} `json:"coordinate" bson:"coordinate"`
}

type ShortestRoute struct {
	GeocodedWaypoints []struct {
		GeocoderStatus string   `json:"geocoder_status"`
		PlaceID        string   `json:"place_id"`
		Types          []string `json:"types"`
	} `json:"geocoded_waypoints"`
	Routes []struct {
		WaypointOrder []int `json:"waypoint_order"`
	} `json:"routes"`
	Status string `json:"status"`
}

var PutRequestIndex int

func getSession() *mgo.Session {

	s, err := mgo.Dial("mongodb://admin:admin@ds043714.mongolab.com:43714/tripplanner")
	if err != nil {
		panic(err)
	}
	return s
}

type UserController struct {
	session *mgo.Session
}

func NewUserController(s *mgo.Session) *UserController {
	return &UserController{s}
}

func createLoc(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	uc := NewUserController(getSession())
	var U UberPrices
	var R Response
	var V Request

	V_temp := MongoResponse{}

	json.NewDecoder(r.Body).Decode(&V)
	json.NewDecoder(r.Body).Decode(&R)

	routes := append(R.Bestlocation_ids, V.StartPt)

	for _, each := range V.Location_ids {

		routes = append(routes, each)
	}

	fmt.Println("optimized routes")
	fmt.Println(routes)
	var Costs int
	var Duration int
	var Distance float64

	Adjacency_matrix := make([][]int, len(routes))
	Distance_matrix := make([][]float64, len(routes))
	Duration_matrix := make([][]int, len(routes))
	for i := 0; i < len(routes); i++ {
		Adjacency_matrix[i] = make([]int, len(routes))
		Distance_matrix[i] = make([]float64, len(routes))
		Duration_matrix[i] = make([]int, len(routes))

		for j := 0; j < len(routes); j++ {

			oid := bson.ObjectIdHex(routes[i])
			if err := uc.session.DB("tripplanner").C("locationsC").FindId(oid).One(&V_temp); err != nil {
				rw.WriteHeader(404)
				return
			}
			UrlParams := "start_latitude=" + strconv.FormatFloat(V_temp.Coordinate.Lat, 'f', -1, 64) + "&start_longitude=" + strconv.FormatFloat(V_temp.Coordinate.Lng, 'f', -1, 64)
			oid = bson.ObjectIdHex(routes[j])

			if err := uc.session.DB("tripplanner").C("locationsC").FindId(oid).One(&V_temp); err != nil {
				rw.WriteHeader(404)
				return
			}

			UrlParams = UrlParams + "&end_latitude=" + strconv.FormatFloat(V_temp.Coordinate.Lat, 'f', -1, 64) + "&end_longitude=" + strconv.FormatFloat(V_temp.Coordinate.Lng, 'f', -1, 64)
			//UrlParams = UrlParams + "&access_token=<HIDDEN>"

			Url := "https://sandbox-api.uber.com/v1/estimates/price?server_token=F7FuxUeOTmYNrr6wFZxVMERWwAcX7zcp5GmxqOqh&" + UrlParams
			fmt.Println("Complete url")
			fmt.Println(Url)
			res, _ := http.Get(Url)
			data, _ := ioutil.ReadAll(res.Body)
			res.Body.Close()
			_ = json.Unmarshal(data, &U)
			for _, each := range U.Prices {
				if each.DisplayName == "uberX" {
					Costs = each.LowEstimate
					Duration = each.Duration
					Distance = each.Distance
				}

			}

			fmt.Println("Cost of travelling from ", i, " to ", j, "is ", Costs)

			Adjacency_matrix[i][j] = Costs
			Distance_matrix[i][j] = Distance
			Duration_matrix[i][j] = Duration
			fmt.Println("Inside the Matrix")		}


//program is not able to reach this statement
fmt.Println("Out of the Matrix")
	}
	fmt.Println("Neither here")

	for i := 0; i < len(Adjacency_matrix); i++ {
		for j := 0; j < len(Adjacency_matrix); j++ {
			Adjacency_matrix[i][i] = 0
			if Adjacency_matrix[i][j] != Adjacency_matrix[j][i] {
				fmt.Println("Change the values", i, j)

				Adjacency_matrix[j][i] = Adjacency_matrix[i][j]

			}
		}
	}
	fmt.Println("Matrix TSP : ", Adjacency_matrix)
	wayPoints := make([]int, 0)
	wayPoints = append(wayPoints, 0)
	visitedNode := make([]int, len(Adjacency_matrix))
	visitedNode[0] = 1
	Dst := 0
	MinFlag := false
	min := math.MaxInt64
	i := 0
	j := 0
	Counter := 0
	fmt.Println(wayPoints)
	for Counter < len(Adjacency_matrix)-1 {
		min = math.MaxInt64
		for j < len(Adjacency_matrix) {

			if Adjacency_matrix[i][j] > 1 && visitedNode[j] == 0 {

				if min > Adjacency_matrix[i][j] {

					min = Adjacency_matrix[i][j]
					Dst = j
					MinFlag = true

				}

			}

			fmt.Println("Iteration ", j)
			j++

		}

		if MinFlag == true {
			i = Dst
			fmt.Println("Iteration ", i)
			visitedNode[Dst] = 1
			MinFlag = false
		}
		fmt.Println("Iteration ", i, j)
		wayPoints = append(wayPoints, i)
		Counter++
		fmt.Println("Counter ", Counter)
		j = 0
	}

	wayPoints = append(wayPoints, 0)
	fmt.Println("Shortest route ", wayPoints)

	Costs = 0
	for a := 0; a < len(wayPoints)-1; a++ {
		fmt.Println(wayPoints[a])
		fmt.Println(wayPoints[a+1])
		i := wayPoints[a]
		j := wayPoints[a+1]

		fmt.Println(i, j, Adjacency_matrix[i][j])
		Costs += Adjacency_matrix[i][j]
		fmt.Println(Costs)
		fmt.Println(Distance_matrix[i][j])
		Distance += Distance_matrix[i][j]
		fmt.Println(Duration_matrix[i][j])
		Duration += Duration_matrix[i][j]

	}

	fmt.Println("Optimized route : ", len(V.Location_ids), len(wayPoints))

	for i := 1; i <= len(wayPoints)-2; i++ {

		fmt.Println("Waypoints", wayPoints[i])
		j := wayPoints[i]
		fmt.Println("Location to be appended", routes[j])
		R.Bestlocation_ids = append(R.Bestlocation_ids, routes[j])
		fmt.Println("Optimized route : ", j, R.Bestlocation_ids)
	}

	fmt.Println("Total costs  is : ", Costs)
	fmt.Println("Total distance is : ", Distance)
	fmt.Println("Total duration is : ", Duration)

	c := uc.session.DB("tripplanner").C("locationsC")
	results := []MongoResponse{}
	_ = c.Find(nil).All(&results)

	if len(results) == 0 {
		R.TripId = strconv.Itoa(12345)
	} else {
		R.TripId = strconv.Itoa(12345 + len(results))
	}

	R.Status = "Planning"
	R.StartPt = V.StartPt
	R.Costs = Costs
	R.Duration = Duration
	R.Distance = Distance

	uc.session.DB("tripplanner").C("locationsC").Insert(R)
	UJ, _ := json.Marshal(R)
	fmt.Fprintf(rw, "%s", UJ)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)

	fmt.Println("End")
}

func getLoc(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	uc := NewUserController(getSession())
	id := p.ByName("id")
	var V Response

	if err := uc.session.DB("tripplanner").C("locationsC").FindId(id).One(&V); err != nil {
		w.WriteHeader(404)
		return
	}
	fmt.Println(V)
	Uj, _ := json.Marshal(V)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", Uj)

}

func putLoc(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	uc := NewUserController(getSession())
	id := p.ByName("id")

	var V Response
	var V_Updated PutResponse
	var P UberEstimates
	//var V_temp Response
	var M MongoResponse
	var ETA int

	if err := uc.session.DB("tripplanner").C("locationsC").FindId(id).One(&V); err != nil {
		w.WriteHeader(404)
		return
	}

	fmt.Println(PutRequestIndex, len(V.Bestlocation_ids))

	if PutRequestIndex < len(V.Bestlocation_ids) {

		Mid := V.Bestlocation_ids[PutRequestIndex]

		if !bson.IsObjectIdHex(Mid) {
			w.WriteHeader(404)
			return
		}

		oid := bson.ObjectIdHex(Mid)

		fmt.Println("REquesting", Mid)

		if err := uc.session.DB("tripplanner").C("locationsC").FindId(oid).One(&M); err != nil {
			w.WriteHeader(404)
			return
		}
		fmt.Println("MongoResponse is ", M)
		Url := "https://sandbox-api.uber.com/v1/estimates/time?start_latitude="
		Url = Url + strconv.FormatFloat(M.Coordinate.Lat, 'f', -1, 64)
		Url = Url + "&start_longitude="
		Url = Url + strconv.FormatFloat(M.Coordinate.Lng, 'f', -1, 64)
//		Url = Url + "&access_token=<NOT DISCLOSED FOR SECURITY REASON.>"

		fmt.Println(Url)
		res, _ := http.Get(Url)
		data, _ := ioutil.ReadAll(res.Body)
		fmt.Println(res.Body)
		res.Body.Close()
		_ = json.Unmarshal(data, &P)
		fmt.Println(P)
		for _, each := range P.Times {
			if each.LocalizedDisplayName == "uberX" {
				ETA = each.Estimate
			}

		}
		var V_temp Response

		json.NewDecoder(r.Body).Decode(&V_temp)

		fmt.Println("Response Id found", V_temp)

		V_Updated.TripId = V.TripId
		V_Updated.StartPt = V.StartPt
		V_Updated.Bestlocation_ids = V.Bestlocation_ids
		V_Updated.Costs = V.Costs
		V_Updated.Duration = V.Duration
		V_Updated.Distance = V.Distance

		V_Updated.Status = V_temp.Status
		V_Updated.ETA = ETA
		V_Updated.NextPt = V.Bestlocation_ids[PutRequestIndex]
		fmt.Println("Updated Response is ", V_Updated)
		PutRequestIndex++
		if err := uc.session.DB("tripplanner").C("locationsC").Update(V, V_Updated); err != nil {
			w.WriteHeader(404)
			return

		}
		uj, _ := json.Marshal(V)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, "%s", uj)

	} else if PutRequestIndex >= len(V.Bestlocation_ids) {
		Msg:= "You have reached the destination"
		PutRequestIndex = 0
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, "%s", Msg)
	}
}

func main() {

	mux := httprouter.New()

	mux.GET("/trips/:id", getLoc)

	mux.POST("/trips", createLoc)

	mux.PUT("/trips/:id/request", putLoc)

	server := http.Server{
		Addr:    "0.0.0.0:3021",
		Handler: mux,
	}
	server.ListenAndServe()
}
