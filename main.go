package main

import (
	"fmt"
	"os"
	"encoding/base64"
    "encoding/json"
    "log"
    "math"
    "net/http"
	"io/ioutil"
    "github.com/gorilla/mux"
    "github.com/parnurzeal/gorequest"
    "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type TrackIDs struct {
	Items []struct {
		Track struct {
			Id string `json:"id"`
			Name string
			Artists []struct {
				Name string
			} `json: "artists"`
			Album struct {
				Name string
			} `json: "artists"`
			Uri string
		} `json:"track"`
	} `json:"items"`
	Next string `json:"next"`
	Total int `json:"total"`
}

type PlaylistListingIncoming struct {
	Items []struct {
		External_urls struct {
			Spotify string
		}
		Name string
		Public bool
		Tracks struct {
			Total int
		}
		Owner struct {
			Id string
		}
	}
	Next string `json:"next"`
}

type PlaylistListingOutgoing struct {
	Tracks []*Track
	Tempo_min float64
	Tempo_max float64

}

type Playlist struct {
	URL string
	Name string
	Public string
	TrackCount int
	OwnerId string
}

type Track struct {
	Id string
	Name string
	Artist string
	Album string
	Acousticness float64
	Danceability float64
	Duration_ms int
	Energy float64
	Instrumentalness float64
	Key int
	Liveness float64
	Loudness float64
	Mode int
	Tempo float64
	Time_signature int
	Valence float64
	Active bool
	Uri string
}

type TrackFeatures struct {
	Audio_features []struct {
		Acousticness float64
		Danceability float64
		Duration_ms int
		Energy float64
		Id string
		Instrumentalness float64
		Key int
		Liveness float64
		Loudness float64
		Mode int
		Tempo float64
		Time_signature int
		Valence float64
	}
}

type AuthorizationResponse struct {
	Refresh_token string
	Access_token string
}

type AuthorizationReturn struct {
	UserID string
	RefreshToken string
}

type UserResponse struct {
	Id string
}

type PlaylistRequest struct {
	Uris []string `json:"uris"`
}

var baseURL string = "https://api.spotify.com"
var clientID string = base64.StdEncoding.EncodeToString([]byte(os.Getenv("SPOTIFY_AUTH")))

func AuthorizeEndpoint(code string)(body string) {
	authURL := "https://accounts.spotify.com/api/token"
	var authString string = "Basic " + clientID;
	var codeString string = "code=" + code

	var redirect_uri string
	if (len(os.Args) > 1 && os.Args[1] == "local") {
		redirect_uri = "http://localhost:3000"
	} else {
		redirect_uri = "https://kiranthawardas.github.io/bandpass"
	}


	request := gorequest.New()
	resp, _, errs := request.Post(authURL).
	  Set("Authorization", authString).
	  Set("Content-Type", "application/x-www-form-urlencoded").
	  Send("grant_type=authorization_code&" + codeString + "&redirect_uri=" + redirect_uri).
	  End()
	if errs != nil {
		panic("Want stack trace")
	}
    respBody, _ := ioutil.ReadAll(resp.Body)

    var authorizationMap AuthorizationResponse
    jsonErr := json.Unmarshal(respBody, &authorizationMap)
	if (jsonErr != nil) {
		fmt.Println(jsonErr)
	}

	var returnValue AuthorizationReturn
	returnValue.RefreshToken = authorizationMap.Refresh_token
	authString = authorizationMap.Access_token

	request = gorequest.New()
	resp, _, errs = request.Get(baseURL + "/v1/me").
		Set("Authorization", "Bearer " + authString).
		End()
	if errs != nil {
		panic("Want stack trace")
	}
	respBody, _ = ioutil.ReadAll(resp.Body)

	var userMap UserResponse
    jsonErr = json.Unmarshal(respBody, &userMap)
	if (jsonErr != nil) {
		fmt.Println(jsonErr)
	}
	returnValue.UserID = userMap.Id
	retString, _ := json.Marshal(returnValue)
	return string(retString)
}

func SpotifyAuthorization(code string) (authToken string) {
	authURL := "https://accounts.spotify.com/api/token"
	var authString string = "Basic " + clientID;
	var codeString string = "refresh_token=" + code

	var redirect_uri string
	if (len(os.Args) > 1 && os.Args[1] == "local") {
		redirect_uri = "http://localhost:3000"
	} else {
		redirect_uri = "https://kiranthawardas.github.io/bandpass"
	}

	request := gorequest.New()
	resp, _, errs := request.Post(authURL).
	  Set("Authorization", authString).
	  Set("Content-Type", "application/x-www-form-urlencoded").
	  Send("grant_type=refresh_token&" + codeString + "&redirect_uri=" + redirect_uri).
	  End()
	if errs != nil {
		panic("Want stack trace")
	}
    respBody, _ := ioutil.ReadAll(resp.Body)

    authMap := make(map[string]string)
    err := json.Unmarshal(respBody, &authMap)
	if err != nil {
		fmt.Println("error:", err)
	}

    authToken = authMap["access_token"]
    return
}

func GetPlaylistsEndpoint(code string)(body string) {

	token := SpotifyAuthorization(code)
	var authString string = "Bearer " + token

	output := make([]*Playlist, 0)

	var currentURL string = baseURL + "/v1/me/playlists"
	for currentURL != "" {
		client := &http.Client{}

		req, err := http.NewRequest("GET", currentURL, nil)
		req.Header.Add("Authorization", authString)

		resp, err := client.Do(req)

		respBody, err := ioutil.ReadAll(resp.Body)
		var playlistListing PlaylistListingIncoming
		err = json.Unmarshal([]byte(respBody), &playlistListing)
		if err != nil {
			fmt.Println("error:", err)
		}

		for _,item := range playlistListing.Items {
			var temp string
			if item.Public == true {
				temp = "Public"
			} else {
				temp = "Private"
			}
			if (item.Tracks.Total > 0) {
				output = append(output, &Playlist{
					URL: item.External_urls.Spotify,
					Name: item.Name,
					Public: temp,
					TrackCount: item.Tracks.Total,
					OwnerId: item.Owner.Id})
			}
		}
		currentURL = playlistListing.Next
	}
	retString, _ := json.Marshal(output)
	return string(retString)
}

func GetPlaylistTracksEndpoint(userID string, playlistID string, code string)(returnValue string) {
	token := SpotifyAuthorization(code)

	tracks := make(map[string]*Track)

	var authString string = "Bearer " + token
	var currentURL string = baseURL + "/v1/users/" + userID + "/playlists/" + playlistID + "/tracks?fields=items(track(id,name,artists,album,uri)),next/"

	for currentURL != "" {
		var trackString string = baseURL + "/v1/audio-features/?ids="

		client := &http.Client{}

		req, err := http.NewRequest("GET", currentURL, nil)
		if err != nil {
			fmt.Println("error:", err)
		}

		req.Header.Add("Authorization", authString)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("error:", err)
		}

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("error:", err)
		}

		var responseStruct TrackIDs
		err = json.Unmarshal([]byte(respBody), &responseStruct)
		if err != nil {
			fmt.Println("error:", err)
		}
		currentURL = responseStruct.Next

		for _,item := range responseStruct.Items {
			if (item.Track.Id == "") {
				continue;
			}
			tracks[item.Track.Id] = &Track{Id: item.Track.Id, Name: item.Track.Name, Uri: item.Track.Uri, Artist: item.Track.Artists[0].Name, Album: item.Track.Album.Name, Active: true}
			trackString = trackString + item.Track.Id + ","
		}
		GetTrackFeatures(trackString, tracks, authString)
	}

	playlistListing := &PlaylistListingOutgoing{
		Tracks: make([]*Track, 0, len(tracks)),
		Tempo_max: 0,
		Tempo_min: math.MaxFloat64,
	}

	for k := range tracks {
		playlistListing.Tracks = append(playlistListing.Tracks, tracks[k])
		if (tracks[k].Tempo > playlistListing.Tempo_max) {
			playlistListing.Tempo_max = tracks[k].Tempo
		}
		if (tracks[k].Tempo < playlistListing.Tempo_min) {
			playlistListing.Tempo_min = tracks[k].Tempo
		}

	}
	if (playlistListing.Tempo_min == math.MaxFloat64) {
		playlistListing.Tempo_min = 0
	}

	retString, _ := json.Marshal(playlistListing)
	return string(retString)

}

func GetTrackFeatures(trackString string, tracks map[string]*Track, authString string) {
	var features TrackFeatures

	client := &http.Client{}

	req, err := http.NewRequest("GET", trackString, nil)
	if err != nil {
		fmt.Println("error:", err)
	}

	req.Header.Add("Authorization", authString)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error:", err)
	}

    respBody, _ := ioutil.ReadAll(resp.Body)

    err = json.Unmarshal([]byte(respBody), &features)
	if err != nil {
		fmt.Println("error:", err)
	}

    for _,item := range features.Audio_features {
    	if (item.Id == "") {
    		continue;
    	}

		tracks[item.Id].Acousticness = item.Acousticness
		tracks[item.Id].Danceability = item.Danceability
		tracks[item.Id].Duration_ms = item.Duration_ms
		tracks[item.Id].Energy = item.Energy
		tracks[item.Id].Instrumentalness = item.Instrumentalness
		tracks[item.Id].Key = item.Key
		tracks[item.Id].Liveness = item.Liveness
		tracks[item.Id].Loudness = item.Loudness
		tracks[item.Id].Mode = item.Mode
		tracks[item.Id].Tempo = item.Tempo
		tracks[item.Id].Time_signature = item.Time_signature
		tracks[item.Id].Valence = item.Valence
    }
}
//
// func SmartFilterEndpoint(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")
//
// 	playlist := make(map[string]*Track)
//
// 	body, _ := ioutil.ReadAll(r.Body)
// 	err := json.Unmarshal(body, &playlist)
// 	if err != nil {
// 		fmt.Println("error:", err)
// 	}
//
// 	var mean float64 = 0
// 	for _, value := range playlist {
// 		mean += value.Energy
// 	}
// 	mean /= float64(len(playlist))
//
// 	var stdev float64 = 0
// 	for _, value := range playlist {
// 		stdev += math.Pow(value.Energy - mean, 2)
// 	}
// 	stdev /= float64(len(playlist))
// 	stdev = math.Pow(stdev, 0.5)
// 	fmt.Println("Mean: %d", mean)
// 	fmt.Println("Mean: %d", stdev)
//
// 	for key, value := range playlist {
// 		if (value.Energy < (mean - (2*stdev)) || value.Energy > (mean + (2*stdev)) ) {
// 			fmt.Println("%s", value.Id)
// 			playlist[key].Active = false
// 		}
// 	}
//
//     json.NewEncoder(w).Encode(playlist)
// }

func CreatePlaylistEndpoint(userID string, playlistName string, code string, body string)(returnBody string){

	token := SpotifyAuthorization(code)
	var authString string = "Bearer " + token

	var currentURL string = baseURL + "/v1/users/" + userID + "/playlists"
	var requestString string = `{"name":"` + playlistName + `", "public":false}`
	request := gorequest.New()
	resp, _, errs := request.Post(currentURL).
		Set("Authorization", authString).
		Send(requestString).
		End()
	if errs != nil {
		panic("Want stack trace")
	}
	respBody, _ := ioutil.ReadAll(resp.Body)


    playlistRespMap := make(map[string]string)
    _ = json.Unmarshal(respBody, &playlistRespMap)

	currentURL = baseURL + "/v1/users/" +userID + "/playlists/" + playlistRespMap["id"] + "/tracks"
	var playlistRequest PlaylistRequest
    _ = json.Unmarshal([]byte(body), &playlistRequest)

	for i := 0; i < len(playlistRequest.Uris); i+=100 {
		var current PlaylistRequest
		if i + 100 < len(playlistRequest.Uris) {
			current.Uris = playlistRequest.Uris[i: i+100]
		} else {
			current.Uris = playlistRequest.Uris[i:]
		}
		byteString, _ := json.Marshal(current)
		request = gorequest.New()
		resp, _, errs = request.Post(currentURL).
			Set("Authorization", authString).
			Send(string(byteString[:])).
			End()
		if errs != nil {
			panic("Want stack trace")
		}
		respBody, _ = ioutil.ReadAll(resp.Body)
	}
	return ""

}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	header  := make(map[string]string)
	header["Access-Control-Allow-Origin"] = "*"
	header["Access-Control-Allow-Methods"] = "GET,HEAD,OPTIONS,POST,PUT"
	header["Access-Control-Allow-Headers"] = "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers"
	var body string
	if (request.Resource == "/bandpass/authorize") {
		body = AuthorizeEndpoint(request.QueryStringParameters["code"])
	} else if (request.Resource == "/bandpass/getplaylisttracks") {
		body = GetPlaylistTracksEndpoint(request.QueryStringParameters["userID"], request.QueryStringParameters["playlistID"], request.QueryStringParameters["code"])
	} else if (request.Resource == "/bandpass/createplaylist") {
		body = CreatePlaylistEndpoint(request.QueryStringParameters["userID"], request.QueryStringParameters["playlistName"], request.QueryStringParameters["code"], request.Body)
	} else if (request.Resource == "/bandpass/getplaylists") {
		body = GetPlaylistsEndpoint(request.QueryStringParameters["code"])
	} else {
		return events.APIGatewayProxyResponse{
	      Body:     "Endpoint not found",
	      StatusCode: 404,
	     }, nil
	}
	return  events.APIGatewayProxyResponse{
		Headers: header,
		Body:     body,
		StatusCode: 200,
	}, nil
}

func localHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT")
	w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")

	requestBody, _ := ioutil.ReadAll(req.Body)
	reqBody := string(requestBody)
	params := mux.Vars(req)
	var body string
	if (params["endpoint"] == "authorize") {
		body = AuthorizeEndpoint(req.FormValue("code"))
	} else if (params["endpoint"] == "getplaylisttracks") {
		body = GetPlaylistTracksEndpoint(req.FormValue("userID"), req.FormValue("playlistID"), req.FormValue("code"))
	} else if (params["endpoint"] == "createplaylist") {
		body = CreatePlaylistEndpoint(req.FormValue("userID"), req.FormValue("playlistName"), req.FormValue("code"), reqBody)
	} else if (params["endpoint"] == "getplaylists") {
		body = GetPlaylistsEndpoint(req.FormValue("code"))
	} else {
		fmt.Println("ERROR")
		fmt.Fprintf(w, body)
	}
	fmt.Fprintf(w, body)
}

func main() {
	if (len(os.Args) > 1) {
		if (os.Args[1] == "local") {
		    router := mux.NewRouter()
		    router.HandleFunc("/bandpass/{endpoint}", localHandler)
		    log.Fatal(http.ListenAndServe(":12345", router))
		}
	} else {
    	lambda.Start(Handler)
	}
}
