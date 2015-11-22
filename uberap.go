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

type PriceEstimate struct {
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

type TimeEstimate struct {
	Times []struct {
		LocalizedDisplayName string `json:"localized_display_name"`
		Estimate             int    `json:"estimate"`
		DisplayName          string `json:"display_name"`
		ProductID            string `json:"product_id"`
	} `json:"times"`
}

type Request struct {
	StartingLoc      string   `json:"starting_from_location_id"`
	Location_ids []string `json:"location_ids"`
}

type Response struct {
	TripId           string   `json:"id" bson:"_id"`
	Status           string   `json:"status" bson:"status"`
	StartingLoc          string   `json:"starting_from_location_id" bson:"starting_from_location_id"`
	Bestlocation_ids []string `json:"best_route_location_ids" bson:"best_route_location_ids"`
	Costs            int      `json:"total_uber_costs" bson:"total_uber_costs"`
	Duration         int      `json:"total_uber_duration" bson:"total_uber_duration"`
	Distance         float64  `json:"total_distance" bson:"total_distance"`
}

type PutResponse struct {
	TripId           string   `json:"id" bson:"_id"`
	Status           string   `json:"status" bson:"status"`
	StartingLoc          string   `json:"starting_from_location_id" bson:"starting_from_location_id"`
	NextPt           string   `json:"next_destination_location_id" bson:"next_destination_location_id"`
	Bestlocation_ids []string `json:"Bestlocation_ids" bson:"Bestlocation_ids"`
	Costs            int      `json:"total_uber_costs" bson:"total_uber_costs"`
	Duration         int      `json:"total_uber_duration" bson:"total_uber_duration"`
	Distance         float64  `json: "total_distance" bson:"total_distance"`
	ETA              int      `json: "uber_wait_time_eta" bson:"uber_wait_time_eta"`
}

type NavResp struct {
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

type Navigator struct {
	session *mgo.Session
}

func LocationNav(s *mgo.Session) *Navigator {
	return &Navigator{s}
}

func Post_trip(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	uc := LocationNav(getSession())
	var Prest PriceEstimate
	var Presp Response
	var OResp Request

	Mgotemp := NavResp{}

	json.NewDecoder(r.Body).Decode(&OResp)
	json.NewDecoder(r.Body).Decode(&Presp)

	routes := append(Presp.Bestlocation_ids, OResp.StartingLoc)

	for _, each := range OResp.Location_ids {

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
			if err := uc.session.DB("tripplanner").C("locationsC").FindId(oid).One(&Mgotemp); err != nil {
				rw.WriteHeader(404)
				return
			}
			UrlParams := "start_latitude=" + strconv.FormatFloat(Mgotemp.Coordinate.Lat, 'f', -1, 64) + "&start_longitude=" + strconv.FormatFloat(Mgotemp.Coordinate.Lng, 'f', -1, 64)
			oid = bson.ObjectIdHex(routes[j])

			if err := uc.session.DB("tripplanner").C("locationsC").FindId(oid).One(&Mgotemp); err != nil {
				rw.WriteHeader(404)
				return
			}

			UrlParams = UrlParams + "&end_latitude=" + strconv.FormatFloat(Mgotemp.Coordinate.Lat, 'f', -1, 64) + "&end_longitude=" + strconv.FormatFloat(Mgotemp.Coordinate.Lng, 'f', -1, 64)


			Url := "https://sandbox-api.uber.com/v1/estimates/price?server_token=F7FuxUeOTmYNrr6wFZxVMERWwAcX7zcp5GmxqOqh&" + UrlParams
			fmt.Println("Complete url")
			fmt.Println(Url)
			res, _ := http.Get(Url)
			data, _ := ioutil.ReadAll(res.Body)
			res.Body.Close()
			_ = json.Unmarshal(data, &Prest)
			for _, each := range Prest.Prices {
				if each.DisplayName == "uberX" {
					Costs = each.HighEstimate
					Duration = each.Duration
					Distance = each.Distance
				}

			}

			fmt.Println("Cost of travelling from ", i, " to ", j, "is ", Costs)

			Adjacency_matrix[i][j] = Costs
			Distance_matrix[i][j] = Distance
			Duration_matrix[i][j] = Duration
			fmt.Println("Inside the Matrix")		}



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

	fmt.Println("Optimized route : ", len(OResp.Location_ids), len(wayPoints))

	for i := 1; i <= len(wayPoints)-2; i++ {

		fmt.Println("Waypoints", wayPoints[i])
		j := wayPoints[i]
		fmt.Println("Location to be appended", routes[j])
		Presp.Bestlocation_ids = append(Presp.Bestlocation_ids, routes[j])
		fmt.Println("Optimized route : ", j, Presp.Bestlocation_ids)
	}

	fmt.Println("Total costs  is : ", Costs)
	fmt.Println("Total distance is : ", Distance)
	fmt.Println("Total duration is : ", Duration)

	c := uc.session.DB("tripplanner").C("locationsC")
	results := []NavResp{}
	_ = c.Find(nil).All(&results)

	if len(results) == 0 {
		Presp.TripId = strconv.Itoa(12345)
	} else {
		Presp.TripId = strconv.Itoa(12345 + len(results))
	}

	Presp.Status = "Planning"
	Presp.StartingLoc = OResp.StartingLoc
	Presp.Costs = Costs
	Presp.Duration = Duration
	Presp.Distance = Distance

	uc.session.DB("tripplanner").C("locationsC").Insert(Presp)
	UJ, _ := json.Marshal(Presp)
	fmt.Fprintf(rw, "%s", UJ)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)

	fmt.Println("End")
}

func Get_trip(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	uc := LocationNav(getSession())
	id := p.ByName("id")
	var OResp Response

	if err := uc.session.DB("tripplanner").C("locationsC").FindId(id).One(&OResp); err != nil {
		w.WriteHeader(404)
		return
	}
	fmt.Println(OResp)
	Uj, _ := json.Marshal(OResp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", Uj)

}

func Put_trip(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	uc := LocationNav(getSession())
	id := p.ByName("id")

	var OResp Response
	var R_itinerary PutResponse
	var P TimeEstimate
	//var Mgotemp Response
	var M NavResp
	var ETA int

	if err := uc.session.DB("tripplanner").C("locationsC").FindId(id).One(&OResp); err != nil {
		w.WriteHeader(404)
		return
	}

	fmt.Println(PutRequestIndex, len(OResp.Bestlocation_ids))

	if PutRequestIndex < len(OResp.Bestlocation_ids) {

		Mid := OResp.Bestlocation_ids[PutRequestIndex]

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
		fmt.Println("NavResp is ", M)
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
		var Mgotemp Response

		json.NewDecoder(r.Body).Decode(&Mgotemp)

		fmt.Println("Response Id found", Mgotemp)

		R_itinerary.TripId = OResp.TripId
		R_itinerary.StartingLoc = OResp.StartingLoc
		R_itinerary.Bestlocation_ids = OResp.Bestlocation_ids
		R_itinerary.Costs = OResp.Costs
		R_itinerary.Duration = OResp.Duration
		R_itinerary.Distance = OResp.Distance

		R_itinerary.Status = Mgotemp.Status
		R_itinerary.ETA = ETA
		R_itinerary.NextPt = OResp.Bestlocation_ids[PutRequestIndex]
		fmt.Println("Updated Response is ", R_itinerary)
		PutRequestIndex++
		if err := uc.session.DB("tripplanner").C("locationsC").Update(OResp, R_itinerary); err != nil {
			w.WriteHeader(404)
			return

		}
		uj, _ := json.Marshal(OResp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, "%s", uj)

	} else if PutRequestIndex >= len(OResp.Bestlocation_ids) {
		Msg:= "You have reached the destination"
		PutRequestIndex = 0
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, "%s", Msg)
	}
}

func main() {

	mux := httprouter.New()

	mux.GET("/trips/:id", Get_trip)

	mux.POST("/trips", Post_trip)

	mux.PUT("/trips/:id/request", Put_trip)

	server := http.Server{
		Addr:    "0.0.0.0:3021",
		Handler: mux,
	}
	server.ListenAndServe()
}
/* Outputs Taken at different times of the day

curl -H "Content-Type: application/json" -X POST -d '{"starting_from_location_id": "56517ac40a956e220c54cc26","location_ids" : [ "565030b20a956e1034831e08", "565030560a956e1034831e07", "5631caf90a956e12440ccdaa", "564d93b90a956e295825d7f4" ]
}' http://localhost:3021/trips
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   500  100   310  100   190     11      6  0:00:31  0:00:27  0:00:04    29{"id":"12377","status":"Planning","starting_from_location_id":"56517ac40a956e220c54cc26","best_route_location_ids":["5631caf90a956e12440ccdaa","564d93b90a956e295825d7f4","565030b20a956e1034831e08","565030560a956e1034831e07"],"total_uber_costs":57,"total_uber_duration":3256,"total_distance":21.470000000000002}


Onkar@onkar-personal MINGW64 /
$ curl -H "Content-Type: application/json" -X POST -d '{"starting_from_location_id": "564e7e740a956e18f0fc6ae7","location_ids" : [ "565030b20a956e1034831e08", "565030560a956e1034831e07", "5631caf90a956e12440ccdaa", "564d93b90a956e295825d7f4" ]
}' http://localhost:3021/trips
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   484  100   294  100   190     10      6  0:00:31  0:00:28  0:00:03    26{"id":"12378","status":"Planning","starting_from_location_id":"564e7e740a956e18f0fc6ae7","best_route_location_ids":["564d93b90a956e295825d7f4","5631caf90a956e12440ccdaa","565030b20a956e1034831e08","565030560a956e1034831e07"],"total_uber_costs":56,"total_uber_duration":3129,"total_distance":17}


*********************0428HRS****************
curl -H "Content-Type: application/json" -X POST -d '{"starting_from_location_id": "564e7e740a956e18f0fc6ae7","location_ids" : [ "565030b20a956e1034831e08", "565030560a956e1034831e07", "5631caf90a956e12440ccdaa", "564d93b90a956e295825d7f4" ]
}' http://localhost:3021/trips
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   484  100   294  100   190      9      6  0:00:32  0:00:31  0:00:01    24{"id":"12381","status":"Planning","starting_from_location_id":"564e7e740a956e18f0fc6ae7","best_route_location_ids":["564d93b90a956e295825d7f4","5631caf90a956e12440ccdaa","565030b20a956e1034831e08","565030560a956e1034831e07"],"total_uber_costs":71,"total_uber_duration":3129,"total_distance":17}


*/
